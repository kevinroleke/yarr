Array.from(document.getElementsByTagName('template')).forEach(function (ele, i) {
    var clon = ele.content.cloneNode(true);
    var desc = document.getElementsByClassName('description')[i];
    desc.appendChild(clon);
});