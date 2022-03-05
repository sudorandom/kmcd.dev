
function randId() {
     return Math.random().toString(36).replace(/[^a-z]+/g, '').substr(2, 10);
}

module.exports = exports = function renderer({ Marked, _relativeURL, _ID }) {

  Marked.image = ( href, title, text ) => {
    let id = randId()
    let out = `<a href="${ href }" class="has-popup-image"><img src="${ href }" id="${ id }" class="center" alt="${ text }"`

    if( title ) {
      out += ` title="${ title }"`;
    }

    out += '></a>';

    if (href.endsWith(".svg")) {
      out +=`<script>var canvas = SVG().addTo('#${ id }').size(2000, 1000).panZoom()</script>`
    }

    return out;
  }

  return Marked;
};
