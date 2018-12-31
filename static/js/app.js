// ##################################################################
// Uploader class
// ##################################################################
class Uploader {
    constructor() {
        this.req = new XMLHttpRequest;
    }

    abort() {
        this.req.abort();
        hide(app.bottomPanel);
    }

    upload(file) {
        document.getElementById("upload-file-name").innerText = file.name;
        show(app.bottomPanel);
        var progress = document.getElementById("progress");

        this.req.upload.onprogress = function(e) {
            progress.innerText = Math.round((e.loaded*100)/e.total) + "%";
        }

        this.req.onload = function() {
            if (this.status == 200) {
                notifier.showMessage("File \"" + file.name + "\" uploaded");
            } else {
                notifier.showMessage("Error: " + this.statusText);
            }
    
            hide(app.bottomPanel);
        }

        this.req.onerror = function () {
            notifier.showMessage("Connection error: check your network connection.");
        }

        var path = "";
        if (app.location.innerText == "/") {
            path = "/upload/" + file.name;
        } else {
            path = "/upload" + app.location.innerText + "/" + file.name;
        }

        this.req.open("POST", path, true);
        this.req.send(file);
    }
}

var uploader = new Uploader();

// ##################################################################
// Notifier class
// ##################################################################
class Notifier {
    constructor() {
        this.messageArea = document.querySelector(".modal-content p");
        this.modalBox = document.getElementById("modal");

        var self = this;

        document.querySelector(".modal-content .flat-button").addEventListener("click", function () {
            self.modalBox.style.display = "none";
        });

        window.addEventListener("click", function (e) {
            if (e.target == self.modalBox) {
                self.modalBox.style.display = "none";
            }
        });
    }

    showMessage(text) {
        this.messageArea.innerText = text;
        this.modalBox.style.display = "initial";
    }
}

var notifier = new Notifier();

// ##################################################################
// FilesList class
// ##################################################################
class FilesList {
    constructor() {
        this.files = document.getElementById("files-list");

        this.files.addEventListener("click", function(e) {
            var target = e.target;
            while (target != this) {
                if (target.matches("a.folder")) {
                    e.preventDefault();
                    app.readDir(target.getAttribute("href"));
                    return;
                }
                target = target.parentNode;
            }
        });
    }

    append(listItem) {
        this.files.appendChild(listItem);
    }

    clear() {
        while (this.files.firstChild) {
            this.files.removeChild(this.files.firstChild);
        }
    }
}

// ##################################################################
// Application singleton object
// ##################################################################
var app = {
    backButton: document.getElementById("back-button"),
    refreshButton: document.getElementById("refresh-button"),
    location: document.getElementById("location"),
    fileinput: document.getElementById("file-input"),
    // searchInput: document.getElementById("search-input"),
    bottomPanel: document.getElementById("bottom-panel"),
    filesList: new FilesList(),
    camera: document.getElementById("camera")
}

app.fileinput.addEventListener("change", handleFileInput);
app.camera.addEventListener("change", handleFileInput);

function handleFileInput(e) {
    var file = e.currentTarget.files[0];
    if (file) { uploader.upload(file) }
    e.currentTarget.value = "";
}

app.goBack = function () {
    app.readDir(dir(app.location.innerText));
};

app.refresh = function () {
    app.readDir(app.location.innerText);
}

app.location.addEventListener("click", function() {
    notifier.showMessage(this.innerText);
})
app.backButton.addEventListener("click", app.goBack);
app.refreshButton.addEventListener("click", app.refresh);

app.readDir = function (dirname) {
    var req = new XMLHttpRequest();

    req.onload = function () {
        if (this.status == 200) {
            app.location.innerText = dirname;
            app.backButton.disabled = (app.location.innerText == "/");
            app.filesList.clear();
            var fileInfoList = JSON.parse(req.responseText);

            if (fileInfoList) {
                for (var i = 0; i < fileInfoList.length; i++) {
                    app.filesList.append(makeListItem(fileInfoList[i]));
                }
            }

        } else {
            notifier.showMessage(req.responseText);
        }
    }

    req.onerror = function () {
        notifier.showMessage("Connection error: check your network connection.");
    }

    req.open("GET", "/files"+dirname);
    req.send();
}

// ##################################################################
// Misc
// ##################################################################
function hide(element) {
    element.style.display = "none";
}

function show(element) {
    element.style.display = "initial";
}

function trimPrefix(s, prefix) {
    if (s.startsWith(prefix)) {
        return s.slice(prefix.length)
    }
    return s
}

function dir(path) {
    var i = path.lastIndexOf("/");
    if (i > 0) {
        return path.slice(0, i);
    }
    return "/";
}

var values = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

function sizeFunc(sizeInBytes) {
    var i = 0;;
    while (sizeInBytes >= 1024) {
        sizeInBytes /= 1024;
        i++;
    }
    return sizeInBytes.toFixed(2) + " " + values[i];
}

// ##################################################################
// ListItem
// ##################################################################

function makeListItem(fileinfo) {
    var li = document.createElement("li");

    var a = document.createElement("a");
    a.target = "_blank";
    a.rel = "nofollow noopener";

    var size = document.createElement("div");
    size.className = "size";

    var modtime = new Date(fileinfo.modtime/1000000);

    var date = document.createElement("div");
    date.className = "date";
    date.innerText = modtime.toLocaleDateString() + " " + modtime.toLocaleTimeString();

    var name = document.createElement("div");
    name.className = "name";
    name.innerText = fileinfo.name;

    if (fileinfo.isdir) {
        if (app.location.innerText == "/") {
            a.href = "/" + fileinfo.name;
        } else {
            a.href = app.location.innerText + "/" + fileinfo.name;
        }
        a.className = "folder icon";
        size.innerText = "dir";
    } else {
        if (app.location.innerText == "/") {
            a.href = "download/" + fileinfo.name;
        } else {
            a.href = "download" + app.location.innerText + "/" + fileinfo.name;
        }
        a.className = "file icon";
        size.innerText = sizeFunc(fileinfo.size);
    }

    a.appendChild(size);
    a.appendChild(date);
    a.appendChild(name);
    li.appendChild(a);

    return li;
}
