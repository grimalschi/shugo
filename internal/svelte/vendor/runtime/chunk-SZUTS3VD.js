import {
  is_void
} from "./chunk-6ZQVU4YW.js";
import {
  dynamic_void_element_content,
  has_own_property,
  snippet_without_render_tag,
  store_invalid_shape,
  svelte_element_invalid_this_value
} from "./chunk-GFOPKZ2A.js";

// node_modules/svelte/src/escaping.js
var ATTR_REGEX = /[&"<]/g, CONTENT_REGEX = /[&<]/g;
function escape_html(value, is_attr) {
  let str = String(value ?? ""), pattern = is_attr ? ATTR_REGEX : CONTENT_REGEX;
  pattern.lastIndex = 0;
  let escaped = "", last = 0;
  for (; pattern.test(str); ) {
    let i = pattern.lastIndex - 1, ch = str[i];
    escaped += str.substring(last, i) + (ch === "&" ? "&amp;" : ch === '"' ? "&quot;" : "&lt;"), last = i + 1;
  }
  return escaped + str.substring(last);
}

// node_modules/clsx/dist/clsx.mjs
function r(e) {
  var t, f, n = "";
  if (typeof e == "string" || typeof e == "number") n += e;
  else if (typeof e == "object") if (Array.isArray(e)) {
    var o = e.length;
    for (t = 0; t < o; t++) e[t] && (f = r(e[t])) && (n && (n += " "), n += f);
  } else for (f in e) e[f] && (n && (n += " "), n += f);
  return n;
}
function clsx() {
  for (var e, t, f = 0, n = "", o = arguments.length; f < o; f++) (e = arguments[f]) && (t = r(e)) && (n && (n += " "), n += t);
  return n;
}

// node_modules/svelte/src/internal/shared/attributes.js
var replacements = {
  translate: /* @__PURE__ */ new Map([
    [!0, "yes"],
    [!1, "no"]
  ])
};
function attr(name, value, is_boolean = !1) {
  if (name === "hidden" && value !== "until-found" && (is_boolean = !0), value == null || is_boolean && !value && value !== "") return "";
  let normalized = has_own_property.call(replacements, name) && replacements[name].get(value) || value, assignment = is_boolean ? '=""' : `="${escape_html(normalized, !0)}"`;
  return ` ${name}${assignment}`;
}
function clsx2(value) {
  return typeof value == "object" ? clsx(value) : value ?? "";
}
var whitespace = [...` 	
\r\f \v\uFEFF`];
function to_class(value, hash, directives) {
  var classname = value == null ? "" : "" + value;
  if (hash && (classname = classname ? classname + " " + hash : hash), directives) {
    for (var key of Object.keys(directives))
      if (directives[key])
        classname = classname ? classname + " " + key : key;
      else if (classname.length)
        for (var len = key.length, a = 0; (a = classname.indexOf(key, a)) >= 0; ) {
          var b = a + len;
          (a === 0 || whitespace.includes(classname[a - 1])) && (b === classname.length || whitespace.includes(classname[b])) ? classname = (a === 0 ? "" : classname.substring(0, a)) + classname.substring(b + 1) : a = b;
        }
  }
  return classname === "" ? null : classname;
}
function append_styles(styles, important = !1) {
  var separator = important ? " !important;" : ";", css = "";
  for (var key of Object.keys(styles)) {
    var value = styles[key];
    value != null && value !== "" && (css += " " + key + ": " + value + separator);
  }
  return css;
}
function to_css_name(name) {
  return name[0] !== "-" || name[1] !== "-" ? name.toLowerCase() : name;
}
function to_style(value, styles) {
  if (styles) {
    var new_style = "", normal_styles, important_styles;
    if (Array.isArray(styles) ? (normal_styles = styles[0], important_styles = styles[1]) : normal_styles = styles, value) {
      value = String(value).replaceAll(/\/\*.*?\*\//g, "").trim();
      var in_str = !1, in_apo = 0, in_comment = !1, reserved_names = [];
      normal_styles && reserved_names.push(...Object.keys(normal_styles).map(to_css_name)), important_styles && reserved_names.push(...Object.keys(important_styles).map(to_css_name));
      var start_index = 0, name_index = -1;
      let len = value.length;
      for (var i = 0; i < len; i++) {
        var c = value[i];
        if (in_comment ? c === "/" && value[i - 1] === "*" && (in_comment = !1) : in_str ? in_str === c && (in_str = !1) : c === "/" && value[i + 1] === "*" ? in_comment = !0 : c === '"' || c === "'" ? in_str = c : c === "(" ? in_apo++ : c === ")" && in_apo--, !in_comment && in_str === !1 && in_apo === 0) {
          if (c === ":" && name_index === -1)
            name_index = i;
          else if (c === ";" || i === len - 1) {
            if (name_index !== -1) {
              var name = to_css_name(value.substring(start_index, name_index).trim());
              if (!reserved_names.includes(name)) {
                c !== ";" && i++;
                var property = value.substring(start_index, i).trim();
                new_style += " " + property + ";";
              }
            }
            start_index = i + 1, name_index = -1;
          }
        }
      }
    }
    return normal_styles && (new_style += append_styles(normal_styles)), important_styles && (new_style += append_styles(important_styles, !0)), new_style = new_style.trim(), new_style === "" ? null : new_style;
  }
  return value == null ? null : String(value);
}

// node_modules/svelte/src/internal/shared/validate.js
function validate_void_dynamic_element(tag_fn) {
  let tag = tag_fn();
  tag && is_void(tag) && dynamic_void_element_content(tag);
}
function validate_dynamic_element_tag(tag_fn) {
  let tag = tag_fn();
  tag && !(typeof tag == "string") && svelte_element_invalid_this_value();
}
function validate_store(store, name) {
  store != null && typeof store.subscribe != "function" && store_invalid_shape(name);
}
function prevent_snippet_stringification(fn) {
  return fn.toString = () => (snippet_without_render_tag(), ""), fn;
}

export {
  escape_html,
  attr,
  clsx2 as clsx,
  to_class,
  to_style,
  validate_void_dynamic_element,
  validate_dynamic_element_tag,
  validate_store,
  prevent_snippet_stringification
};
