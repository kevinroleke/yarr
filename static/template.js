Array.from(document.getElementsByTagName('template')).forEach(function (ele, i) {
    var clon = ele.content.cloneNode(true);
    var desc = document.getElementsByClassName('description')[i];
    desc.appendChild(clon);
});

document.getElementById('pod-search-btn').addEventListener('click', function () {
    window.location.href = '/search/pods/' + document.getElementById('pod-search').value;
});

document.getElementById('pod-search').addEventListener('keydown', function (e) {
    if (e.key == "Enter") window.location.href = '/search/pods/' + document.getElementById('pod-search').value;
});

document.getElementById('ep-search-btn').addEventListener('click', function () {
    window.location.href = '/search/episodes/' + window.PODID + '/' + document.getElementById('ep-search').value;
});

document.getElementById('ep-search').addEventListener('keydown', function (e) {
    if (e.key == "Enter") window.location.href = '/search/episodes/' + window.PODID + '/' + document.getElementById('ep-search').value;
});