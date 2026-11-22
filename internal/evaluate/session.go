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
//	L  list the pages of routes,    P  pages: route \x02 address, split by \x03
//	   split by commas; of all      G  a route is listed: route \x02 whether its
//	   routes if none is named         pages get more than an address \x02 what
//	C  compute: units, split by        the listing read from the data
//	   \x03; a unit is p \x02       R  records, split by \x03
//	   address \x02 route, or       F  a shared block failed: name \x02 the failure
//	   s \x02 name                  S  a shape of readings: number \x02 what it is
//	K  what is known from earlier   E  it failed: { message, stack, place }
//	   runs, once at the start:
//	   { plans, shapes }: see known
//	Q  nothing more
//	I  the index asked for is       I  an index is wanted: table \x02 1 for
//	   there: 1 if it was made         rows in their order, 0 for a count
//	   just now, else 0                \x02 how many columns rows are found
//	                                   by with given values \x02 these
//	                                   columns and those of the order, split
//	                                   by \x02; the answer is waited for
//	J  default search result       J  a search declaration, during startup or evaluation
//	E  a search error              A  ready to receive the first command (search builds)
//	                                D  the request is done: what it has to add
type wire struct {
	asks    io.Writer
	answers *bufio.Reader
	message []byte
	search  func([]byte) ([]byte, error)
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
func (w *wire) ask(kind byte, body []byte, k *keeper, got func(kind byte, body []byte) error) (done []byte, err error) {
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
			failure, err := failed(body)
			if err != nil {
				return nil, err
			}
			return nil, failure
		case 'S':
			// A shape may be told at any moment before a reading of it.
			number, spec, found := bytes.Cut(body, []byte{2})
			at, err := strconv.Atoi(string(number))
			if !found || err != nil || k.shapes == nil {
				return nil, errorf("a damaged shape of readings")
			}
			if err := k.shapes.tell(at, string(spec)); err != nil {
				return nil, err
			}
		case 'J':
			if err := w.searchReply(body); err != nil {
				return nil, err
			}
		case 'I':
			// The project's code waits for the answer before it asks its
			// question: with the index, the question finds its rows at once.
			// It asks only about a question it has not been told of: see known.
			if k.shapes == nil {
				return nil, errorf("a request for an index of a data database, and the project names none")
			}
			p, err := readPlan(string(body))
			if err != nil {
				return nil, err
			}
			made, err := k.index(p)
			if err != nil {
				return nil, err
			}
			answer := []byte("0")
			if made {
				answer = []byte("1")
			}
			if err := w.send('I', answer); err != nil {
				return nil, err
			}
		default:
			if err := got(kind, body); err != nil {
				return nil, err
			}
		}
	}
}

func (w *wire) searchReply(body []byte) error {
	if w.search == nil {
		return errorf("search was not enabled for this build")
	}
	if len(body) > 1<<20 {
		return errorf("a search declaration of %d bytes", len(body))
	}
	result, err := w.search(body)
	if err == nil {
		return w.send('J', result)
	}
	encoded, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{err.Error()})
	return w.send('E', encoded)
}

// Module initializers can use search before run receives its first command.
func (w *wire) ready() error {
	for {
		kind, body, err := w.read()
		if err != nil {
			return err
		}
		switch kind {
		case 'A':
			return nil
		case 'J':
			if err := w.searchReply(body); err != nil {
				return err
			}
		case 'E':
			failure, err := failed(body)
			if err != nil {
				return err
			}
			return failure
		default:
			return errorf("a message of the kind %q before the project's code was ready", kind)
		}
	}
}

// failed reads a failure as the project's code reports it.
func failed(body []byte) (*ScriptError, error) {
	failure := &ScriptError{}
	if err := json.Unmarshal(body, failure); err != nil {
		return nil, fmt.Errorf("the project's code failed, and its report cannot be read: %w", err)
	}
	return failure, nil
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

// list asks which pages the routes have: the ones named, or all of them.
// What the project's code says of every route goes to the keeper.
func (w *wire) list(routes []int, k *keeper) ([]listed, error) {
	var request []byte
	for i, route := range routes {
		if i > 0 {
			request = append(request, ',')
		}
		request = strconv.AppendInt(request, int64(route), 10)
	}
	var pages []listed
	done, err := w.ask('L', request, k, func(kind byte, body []byte) error {
		switch kind {
		case 'P':
			for len(body) > 0 {
				var one []byte
				one, body, _ = bytes.Cut(body, []byte{3})
				number, path, found := bytes.Cut(one, []byte{2})
				route, err := strconv.Atoi(string(number))
				if !found || err != nil || route < 0 || route >= len(k.site.Routes) {
					return errorf("a damaged list of pages")
				}
				if !validPath(string(path)) {
					return errorf("a page with the address %q", path)
				}
				pages = append(pages, listed{route, string(path)})
			}
		case 'G':
			number, reads, found := bytes.Cut(body, []byte{2})
			route, err := strconv.Atoi(string(number))
			if !found || err != nil {
				return errorf("a damaged note of a route")
			}
			return k.route(route, reads)
		default:
			return errorf("a message of the kind %q where the pages were asked for", kind)
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
	k.report.Shadowed += counts.Shadowed
	return pages, nil
}

// compute asks for the records of units and hands them to the keeper. Every
// unit asked for must come back, or be told to have failed.
func (w *wire) compute(units []unit, k *keeper) error {
	if len(units) == 0 {
		return nil
	}
	var request []byte
	for i, u := range units {
		if i > 0 {
			request = append(request, 3)
		}
		if u.shared {
			request = append(append(request, 's', 2), u.name...)
		} else {
			request = append(append(append(request, 'p', 2), u.name...), 2)
			request = strconv.AppendInt(request, int64(u.route), 10)
		}
	}
	before := k.count()
	_, err := w.ask('C', request, k, func(kind byte, body []byte) error {
		switch kind {
		case 'R':
			for len(body) > 0 {
				var one []byte
				one, body, _ = bytes.Cut(body, []byte{3})
				if err := k.put(one); err != nil {
					return err
				}
			}
		case 'F':
			name, report, _ := bytes.Cut(body, []byte{2})
			failure, err := failed(report)
			if err != nil {
				return err
			}
			return k.fail(string(name), failure)
		default:
			return errorf("a message of the kind %q where records were asked for", kind)
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
func start(p *project.Project, log io.Writer, catalog []byte, search *searchPreparation) (*session, error) {
	var catalogInput *os.File
	if len(catalog) > 0 {
		input, err := os.CreateTemp(p.Path(project.Output), ".catalog-input-*")
		if err != nil {
			return nil, err
		}
		defer os.Remove(input.Name())
		defer input.Close()
		if _, err := input.Write(catalog); err != nil {
			return nil, err
		}
		if _, err := input.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		catalogInput = input
	}
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
	catalogFD := ""
	if catalogInput != nil {
		// Module initializers may call search() before the protocol starts.
		// Pass its JSON as a file, since an environment value has a size limit.
		catalogFD = strconv.Itoa(3 + len(command.ExtraFiles))
		command.ExtraFiles = append(command.ExtraFiles, catalogInput)
	}
	searchEnabled := ""
	if search != nil {
		searchEnabled = "1"
	}
	command.Env = append(os.Environ(), "SHUGO_OUT=3", "SHUGO_IN=4", "SHUGO_DATA="+p.Data(), "SHUGO_CATALOG_FD="+catalogFD, "SHUGO_SEARCH="+searchEnabled)
	err = command.Start()
	answersIn.Close()
	asksOut.Close()
	if err != nil {
		answers.Close()
		asks.Close()
		return nil, fmt.Errorf("cannot start %s to run the project's code: %w", p.Runtime(), err)
	}
	session := &session{
		wire:    wire{asks: asks, answers: bufio.NewReaderSize(answers, 1<<20)},
		project: p, command: command, pipes: [2]*os.File{asks, answers},
	}
	if search != nil {
		session.wire.search = search.request
	}
	return session, nil
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

// worker is the JavaScript of the project, started when it is first asked
// for something: an update that finds nothing to compute never starts it.
type worker struct {
	catalog []byte
	search  *searchPreparation
	project *project.Project
	log     io.Writer
	session *session
}

func (w *worker) ready(k *keeper) error {
	if w.session != nil {
		return nil
	}
	// What is known is made sure of before the project's code starts
	// reading: an index made after that would not be seen by it.
	var known []byte
	if k.shapes != nil {
		var err error
		if known, err = k.known(); err != nil {
			return err
		}
	}
	session, err := start(w.project, w.log, w.catalog, w.search)
	if err != nil {
		return err
	}
	w.session = session
	if w.search != nil {
		if err := session.wire.ready(); err != nil {
			return session.told(err)
		}
	}
	if known == nil {
		return nil
	}
	return session.told(session.send('K', known))
}

func (w *worker) list(routes []int, k *keeper) ([]listed, error) {
	if err := w.ready(k); err != nil {
		return nil, err
	}
	pages, err := w.session.wire.list(routes, k)
	return pages, w.session.told(err)
}

func (w *worker) compute(units []unit, k *keeper) error {
	if len(units) == 0 {
		return nil
	}
	if err := w.ready(k); err != nil {
		return err
	}
	return w.session.told(w.session.wire.compute(units, k))
}

// end lets the project's code finish, if it was started.
func (w *worker) end() error {
	if w.session == nil {
		return nil
	}
	return w.session.end()
}

func (w *worker) stop() {
	if w.session != nil {
		w.session.stop()
	}
}

var errDamaged = errors.New("the project's code sent a damaged block")
