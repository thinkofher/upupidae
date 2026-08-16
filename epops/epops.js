/**
 * Custom element that declaratively loads a stylesheet into the document head.
 *
 * The element reads the stylesheet URL from the `href` attribute, adds a
 * corresponding `<link rel="stylesheet">` to `document.head` if it has not
 * already been added, and then removes itself from the DOM.
 *
 * @example
 * <upupidae-style href="/tw/foo/styles.css"></upupidae-style>
 */
class UpupidaeStyle extends HTMLElement {
	connectedCallback() {
		const href = this.getAttribute("href");

		if (!href) return;

		const existing = document.head.querySelector(
			'link[rel="stylesheet"][href="' + CSS.escape(href) + '"]'
		);

		if (!existing) {
			const link = document.createElement("link");
			link.rel = "stylesheet";
			link.href = href;
			document.head.appendChild(link);
		}

		this.remove();
	}
}

customElements.define("upupidae-style", UpupidaeStyle);
