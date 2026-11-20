package evaluate

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/project"
)

// wire is the conversation with the JavaScript of the project. The project's
// code answers requests one at a time: which pages there are, and what the
// records of the pages and shared blocks named to it are.
//
// A message in either direction is a length in four bytes, a letter that
// says what the message is, and its body.
//
//	to the project's code           from it
//	L  list the pages               P  pages: route \x02 address, split by \x03
//	C  compute: units, split by     R  records, split by \x03
//	   \x03; a unit is p or s,      E  it failed: { message, stack, place }
//	   \x02, the address or name    D  the request is done: what it has to add
//	Q  nothing more
type wire struct {
	asks    io.Writer
	answers *bufio.Reader
	message []byte
}

// Limits on what the project's code may send. It runs outside of our
// control, so nothing it sends is taken on trust.
const (
	maxFrame  = 1 << 30 // bytes of one message
	maxBlocks = 1 << 22 // blocks of one page
)

// lost tells that the conversation broke off.
type lost struct{ cause error }

func (l *lost) Error() string {
	return "the project's code stopped answering: " + l.cause.Error()
}

// send writes one message to the project's code.
func (w *wire) send(kind byte, body []byte) error {
	message := binary.LittleEndian.AppendUint32(make([]byte, 0, len(body)+5), uint32(len(body)+1))
	if _, err := w.asks.Write(append(append(message, kind), body...)); err != nil {
		return &lost{err}
	}
	return nil
}

// ask sends one request and hands every message of the answer to got. It
// returns what the project's code added when it was done.
func (w *wire) ask(kind byte, body []byte, got func(kind byte, body []byte) error) (done []byte, err error) {
	if err := w.send(kind, body); err != nil {
		return nil, err
	}
	for {
		kind, body, err := w.read()
		if err != nil {
			return nil, err
		}
		switch kind {
		case 'D':
			return body, nil
		case 'E':
			failure := &ScriptError{}
			if err := json.Unmarshal(body, failure); err != nil {
				return nil, fmt.Errorf("the project's code failed, and its report cannot be read: %w", err)
			}
			return nil, failure
		default:
			if err := got(kind, body); err != nil {
				return nil, err
			}
		}
	}
}

// read takes one message of the project's code.
func (w *wire) read() (kind byte, body []byte, err error) {
	var head [4]byte
	if _, err := io.ReadFull(w.answers, head[:]); err != nil {
		return 0, nil, &lost{err}
	}
	size := int(binary.LittleEndian.Uint32(head[:]))
	if size < 1 || size > maxFrame {
		return 0, nil, errorf("a message of %d bytes", size)
	}
	if cap(w.message) < size {
		w.message = make([]byte, size)
	}
	w.message = w.message[:size]
	if _, err := io.ReadFull(w.answers, w.message); err != nil {
		return 0, nil, &lost{err}
	}
	return w.message[0], w.message[1:], nil
}

// listed is one page the project has.
type listed struct {
	route int // the route that gives it, by its place among the routes of the site
	path  string
}

// list asks which pages the project has.
func (w *wire) list(report *Report) ([]listed, error) {
	var pages []listed
	done, err := w.ask('L', nil, func(kind byte, body []byte) error {
		if kind != 'P' {
			return errorf("a message of the kind %q where the pages were asked for", kind)
		}
		for len(body) > 0 {
			var one []byte
			one, body, _ = bytes.Cut(body, []byte{3})
			number, path, found := bytes.Cut(one, []byte{2})
			route, err := strconv.Atoi(string(number))
			if !found || err != nil || route < 0 {
				return errorf("a damaged list of pages")
			}
			if !validPath(string(path)) {
				return errorf("a page with the address %q", path)
			}
			pages = append(pages, listed{route, string(path)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var counts struct{ Shadowed int }
	if err := json.Unmarshal(done, &counts); err != nil {
		return nil, fmt.Errorf("the project's code sent a damaged summary: %w", err)
	}
	report.Shadowed = counts.Shadowed
	return pages, nil
}

// compute asks for the records of units and hands them to the keeper. Every
// unit asked for must come back.
func (w *wire) compute(units []unit, k *keeper) error {
	if len(units) == 0 {
		return nil
	}
	var request []byte
	for i, u := range units {
		if i > 0 {
			request = append(request, 3)
		}
		kind := byte('p')
		if u.shared {
			kind = 's'
		}
		request = append(append(append(request, kind), 2), u.name...)
	}
	before := k.count()
	_, err := w.ask('C', request, func(kind byte, body []byte) error {
		if kind != 'R' {
			return errorf("a message of the kind %q where records were asked for", kind)
		}
		for len(body) > 0 {
			var one []byte
			one, body, _ = bytes.Cut(body, []byte{3})
			if err := k.put(one); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if got := k.count() - before; got != len(units) {
		return errorf("%d records where %d were asked for", got, len(units))
	}
	for _, u := range units {
		if !k.has(u) {
			return errorf("no record of %s", u)
		}
	}
	return nil
}

// session is the JavaScript of the project while it runs.
type session struct {
	wire
	project *project.Project
	command *exec.Cmd
	pipes   [2]*os.File
}

// start runs the JavaScript of the project.
func start(p *project.Project, log io.Writer) (*session, error) {
	answers, answersIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	asksOut, asks, err := os.Pipe()
	if err != nil {
		answers.Close()
		answersIn.Close()
		return nil, err
	}
	command := exec.Command(p.Runtime(), p.Path(project.Output, compile.Bundle))
	command.Dir = p.Root
	command.Stdout, command.Stderr = log, log
	// Requests and answers go through pipes of their own, so that whatever
	// the project prints or reads cannot get mixed into them.
	command.ExtraFiles = []*os.File{answersIn, asksOut}
	command.Env = append(os.Environ(), "SHUGO_OUT=3", "SHUGO_IN=4")
	err = command.Start()
	answersIn.Close()
	asksOut.Close()
	if err != nil {
		answers.Close()
		asks.Close()
		return nil, fmt.Errorf("cannot start %s to run the project's code: %w", p.Runtime(), err)
	}
	return &session{
		wire:    wire{asks: asks, answers: bufio.NewReaderSize(answers, 1<<20)},
		project: p, command: command, pipes: [2]*os.File{asks, answers},
	}, nil
}

// told turns an error of the conversation into what the author of the
// project needs to know: where their code failed, or how the process ended.
func (s *session) told(err error) error {
	var script *ScriptError
	var broken *lost
	switch {
	case errors.As(err, &script):
		script.locate(s.project)
	case errors.As(err, &broken):
		s.pipes[0].Close()
		if exit := s.command.Wait(); exit != nil {
			return fmt.Errorf("%s ended before the site was computed: %w", s.project.Runtime(), exit)
		}
		return fmt.Errorf("%s ended before the site was computed", s.project.Runtime())
	}
	return err
}

func (s *session) list(report *Report) ([]listed, error) {
	pages, err := s.wire.list(report)
	return pages, s.told(err)
}

func (s *session) compute(units []unit, k *keeper) error {
	return s.told(s.wire.compute(units, k))
}

// end lets the project's code finish.
func (s *session) end() error {
	if err := s.send('Q', nil); err != nil {
		return s.told(err)
	}
	s.pipes[0].Close()
	if exit := s.command.Wait(); exit != nil {
		return fmt.Errorf("%s failed after the site was computed: %w", s.project.Runtime(), exit)
	}
	return nil
}

// stop makes sure the project's code is not left running. After end it
// does nothing.
func (s *session) stop() {
	if s.command.ProcessState == nil {
		s.command.Process.Kill()
		s.command.Wait()
	}
	for _, pipe := range s.pipes {
		pipe.Close()
	}
}

var errDamaged = errors.New("the project's code sent a damaged block")
