
module.exports = exports = function renderer({ Marked, _relativeURL, _ID }) {

  Marked.image = ( href, title, text ) => {
    let out = `<a href="${ href }" class="has-popup-image" data-fancybox-group="markdown-images"><img src="${ href }" class="center" alt="${ text }"`

    if( title ) {
      out += ` title="${ title }"`;
    }

    out += '></a>';

    return out;
  }

  return Marked;
};
