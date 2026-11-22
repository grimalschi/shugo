import {
  false_default,
  transition_slide_display
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/transition/index.js
var linear = (x) => x;
function cubic_out(t) {
  let f = t - 1;
  return f * f * f + 1;
}
function cubic_in_out(t) {
  return t < 0.5 ? 4 * t * t * t : 0.5 * Math.pow(2 * t - 2, 3) + 1;
}
function split_css_unit(value) {
  let split = typeof value == "string" && value.match(/^\s*(-?[\d.]+)([^\s]*)\s*$/);
  return split ? [parseFloat(split[1]), split[2] || "px"] : [
    /** @type {number} */
    value,
    "px"
  ];
}
function css_dimension(property, value, t) {
  return Number.isNaN(value) ? "" : `${property}: ${t * value}px;`;
}
function blur(node, { delay = 0, duration = 400, easing = cubic_in_out, amount = 5, opacity = 0 } = {}) {
  let style = getComputedStyle(node), target_opacity = +style.opacity, f = style.filter === "none" ? "" : style.filter, od = target_opacity * (1 - opacity), [value, unit] = split_css_unit(amount);
  return {
    delay,
    duration,
    easing,
    css: (_t, u) => `opacity: ${target_opacity - od * u}; filter: ${f} blur(${u * value}${unit});`
  };
}
function fade(node, { delay = 0, duration = 400, easing = linear } = {}) {
  let o = +getComputedStyle(node).opacity;
  return {
    delay,
    duration,
    easing,
    css: (t) => `opacity: ${t * o}`
  };
}
function fly(node, { delay = 0, duration = 400, easing = cubic_out, x = 0, y = 0, opacity = 0 } = {}) {
  let style = getComputedStyle(node), target_opacity = +style.opacity, transform = style.transform === "none" ? "" : style.transform, od = target_opacity * (1 - opacity), [x_value, x_unit] = split_css_unit(x), [y_value, y_unit] = split_css_unit(y);
  return {
    delay,
    duration,
    easing,
    css: (t, u) => `
			transform: ${transform} translate(${(1 - t) * x_value}${x_unit}, ${(1 - t) * y_value}${y_unit});
			opacity: ${target_opacity - od * u}`
  };
}
var slide_warning = !1;
function slide(node, { delay = 0, duration = 400, easing = cubic_out, axis = "y" } = {}) {
  let style = getComputedStyle(node);
  false_default && !slide_warning && /(contents|inline|table)/.test(style.display) && (slide_warning = !0, Promise.resolve().then(() => slide_warning = !1), transition_slide_display(style.display));
  let opacity = +style.opacity, primary_property = axis === "y" ? "height" : "width", primary_property_value = parseFloat(style[primary_property]), secondary_properties = axis === "y" ? ["top", "bottom"] : ["left", "right"], capitalized_secondary_properties = secondary_properties.map(
    (e) => (
      /** @type {'Left' | 'Right' | 'Top' | 'Bottom'} */
      `${e[0].toUpperCase()}${e.slice(1)}`
    )
  ), padding_start_value = parseFloat(style[`padding${capitalized_secondary_properties[0]}`]), padding_end_value = parseFloat(style[`padding${capitalized_secondary_properties[1]}`]), margin_start_value = parseFloat(style[`margin${capitalized_secondary_properties[0]}`]), margin_end_value = parseFloat(style[`margin${capitalized_secondary_properties[1]}`]), border_width_start_value = parseFloat(
    style[`border${capitalized_secondary_properties[0]}Width`]
  ), border_width_end_value = parseFloat(
    style[`border${capitalized_secondary_properties[1]}Width`]
  );
  return {
    delay,
    duration,
    easing,
    css: (t) => `overflow: hidden;opacity: ${Math.min(t * 20, 1) * opacity};` + css_dimension(primary_property, primary_property_value, t) + css_dimension(`padding-${secondary_properties[0]}`, padding_start_value, t) + css_dimension(`padding-${secondary_properties[1]}`, padding_end_value, t) + css_dimension(`margin-${secondary_properties[0]}`, margin_start_value, t) + css_dimension(`margin-${secondary_properties[1]}`, margin_end_value, t) + css_dimension(`border-${secondary_properties[0]}-width`, border_width_start_value, t) + css_dimension(`border-${secondary_properties[1]}-width`, border_width_end_value, t) + `min-${primary_property}: 0`
  };
}
function scale(node, { delay = 0, duration = 400, easing = cubic_out, start = 0, opacity = 0 } = {}) {
  let style = getComputedStyle(node), target_opacity = +style.opacity, transform = style.transform === "none" ? "" : style.transform, sd = 1 - start, od = target_opacity * (1 - opacity);
  return {
    delay,
    duration,
    easing,
    css: (_t, u) => `
			transform: ${transform} scale(${1 - sd * u});
			opacity: ${target_opacity - od * u}
		`
  };
}
function draw(node, { delay = 0, speed, duration, easing = cubic_in_out } = {}) {
  let len = node.getTotalLength(), style = getComputedStyle(node);
  return style.strokeLinecap !== "butt" && (len += parseInt(style.strokeWidth)), duration === void 0 ? speed === void 0 ? duration = 800 : duration = len / speed : typeof duration == "function" && (duration = duration(len)), {
    delay,
    duration,
    easing,
    css: (_, u) => `
			stroke-dasharray: ${len};
			stroke-dashoffset: ${u * len};
		`
  };
}
function assign(tar, src) {
  for (let k in src) tar[k] = src[k];
  return (
    /** @type {T & S} */
    tar
  );
}
function crossfade({ fallback, ...defaults }) {
  let to_receive = /* @__PURE__ */ new Map(), to_send = /* @__PURE__ */ new Map();
  function crossfade2(from_node, node, params) {
    let {
      delay = 0,
      duration = (
        /** @param {number} d */
        (d2) => Math.sqrt(d2) * 30
      ),
      easing = cubic_out
    } = assign(assign({}, defaults), params), from = from_node.getBoundingClientRect(), to = node.getBoundingClientRect(), dx = from.left - to.left, dy = from.top - to.top, dw = from.width / to.width, dh = from.height / to.height, d = Math.sqrt(dx * dx + dy * dy), style = getComputedStyle(node), transform = style.transform === "none" ? "" : style.transform, opacity = +style.opacity;
    return {
      delay,
      duration: typeof duration == "function" ? duration(d) : duration,
      easing,
      css: (t, u) => `
			   opacity: ${t * opacity};
			   transform-origin: top left;
			   transform: ${transform} translate(${u * dx}px,${u * dy}px) scale(${t + (1 - t) * dw}, ${t + (1 - t) * dh});
		   `
    };
  }
  function transition(items, counterparts, intro) {
    return (node, params) => (items.set(params.key, node), () => {
      if (counterparts.has(params.key)) {
        let other_node = counterparts.get(params.key);
        return counterparts.delete(params.key), crossfade2(
          /** @type {Element} */
          other_node,
          node,
          params
        );
      }
      return items.delete(params.key), fallback && fallback(node, params, intro);
    });
  }
  return [transition(to_send, to_receive, !1), transition(to_receive, to_send, !0)];
}
export {
  blur,
  crossfade,
  draw,
  fade,
  fly,
  scale,
  slide
};
