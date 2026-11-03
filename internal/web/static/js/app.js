// The app's behaviour. Alpine's CSP build cannot evaluate expressions written
// in the HTML, so every component is defined here with Alpine.data and the
// templates refer to it by name (x-data="counter"). Attributes may name a
// property or call a method, nothing more.
document.addEventListener("alpine:init", () => {
  // The masthead menu on narrow screens.
  Alpine.data("menu", () => ({
    open: false,
    toggle() { this.open = !this.open; },
    get expanded() { return this.open ? "true" : "false"; },
    get menuClass() { return this.open ? "is-open" : ""; },
  }));

  // Characters left in a text box. The limit comes from data-max on the
  // component's own element, read once in init: inside a getter, $el is the
  // element doing the reading, not the component.
  Alpine.data("counter", () => ({
    text: "",
    max: 0,
    init() { this.max = Number(this.$el.dataset.max) || 0; },
    get remaining() {
      const left = this.max - this.text.length;
      return left === 1 ? "1 character left" : left + " characters left";
    },
    clear(event) {
      if (event.detail && event.detail.successful) { this.text = ""; }
    },
  }));
});
