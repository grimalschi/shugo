import {
  cubicOut
} from "./chunk-Q4LEMBIN.js";

// node_modules/svelte/src/animate/index.js
function flip(node, { from, to }, params = {}) {
  var { delay = 0, duration = (d) => Math.sqrt(d) * 120, easing = cubicOut } = params, style = getComputedStyle(node), transform = style.transform === "none" ? "" : style.transform, [ox, oy] = style.transformOrigin.split(" ").map(parseFloat);
  ox /= node.clientWidth, oy /= node.clientHeight;
  var zoom = get_zoom(node), sx = node.clientWidth / to.width / zoom, sy = node.clientHeight / to.height / zoom, fx = from.left + from.width * ox, fy = from.top + from.height * oy, tx = to.left + to.width * ox, ty = to.top + to.height * oy, dx = (fx - tx) * sx, dy = (fy - ty) * sy, dsx = from.width / to.width, dsy = from.height / to.height;
  return {
    delay,
    duration: typeof duration == "function" ? duration(Math.sqrt(dx * dx + dy * dy)) : duration,
    easing,
    css: (t, u) => {
      var x = u * dx, y = u * dy, sx2 = t + u * dsx, sy2 = t + u * dsy;
      return `transform: ${transform} translate(${x}px, ${y}px) scale(${sx2}, ${sy2});`;
    }
  };
}
function get_zoom(element) {
  if ("currentCSSZoom" in element)
    return (
      /** @type {number} */
      element.currentCSSZoom
    );
  for (var current = element, zoom = 1; current !== null; )
    zoom *= +getComputedStyle(current).zoom, current = /** @type {Element | null} */
    current.parentElement;
  return zoom;
}
export {
  flip
};
