let xhr = new XMLHttpRequest();

function submit() {
	var rsakey = document.getElementById("publickey").textContent;

	var s = document.getElementsByTagName("select")[0];
	let kind = s.options[s.selectedIndex].value;
	let code = document.getElementById("code").value;

	var data = encrypt(rsakey, code);

	var url = window.location.href + "/" + kind

	request("POST", url, data, draw);
}

function encrypt(rsakey, code) {
	var array = new Uint32Array(7);
	window.crypto.getRandomValues(array);
	var rand = array.join("");
	var aeskey = rand.slice(0, 32);
	var iv = rand.slice(rand.length - 16, rand.length);

	var core = new JSEncrypt();
	core.setPublicKey(rsakey);
	var cipherkey = core.encrypt(iv + aeskey);

	var ciphercode = CryptoJS.AES.encrypt(code, CryptoJS.enc.Utf8.parse(aeskey), {
		iv: CryptoJS.enc.Utf8.parse(iv),
		mode: CryptoJS.mode.CFB,
		padding: CryptoJS.pad.NoPadding,
	}).toString();

	var data = { "key": cipherkey, "code": ciphercode }

	return data
}

function request(method, url, data, callback) {
	xhr.open(method, url, true);
	xhr.onreadystatechange = function () {
		if (xhr.readyState == xhr.DONE) {
			if (xhr.status == 200) {
				var data = JSON.parse(xhr.responseText);
				if (data["status"]) {
					callback(JSON.parse(data["msg"]));
					errormsg("");
				} else { errormsg(data["msg"]); }
			} else {
				errormsg(xhr.statusText);
			}
		}
	};

	xhr.ontimeout = function () {
		errormsg(xhr.timeout);
	}

	if (method == "POST") {
		xhr.setRequestHeader("Content-Type", "application/json");
		xhr.send(JSON.stringify(data));
	} else {
		xhr.send();
	}
}

function errormsg(msg) {
	var err = document.getElementsByClassName("errormsg")[0];
	err.textContent = msg;
}