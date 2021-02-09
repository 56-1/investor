function standard(x, mean, std) {
    var y = new Float64Array(x.length);
    for (var i = 0; i < x.length; i++) {
        y[i] = calc(x[i], mean, std);
    }

    return y;
}

function calc(value, mean, std) {
    var k = 1.0 / (math.sqrt(2 * math.pi) * std);
    var m = math.exp(-1.0 * math.pow((value - mean), 2.0) / (2 * math.pow(std, 2.0)));
    return k * m;
}

function draw(data) {
    var mean = parseFloat(data["mean"].toFixed(4));
    var std = data["std"];
    var now = data["now"];
    var x = data["poins"];

    var lm = parseFloat((0.3 * std + mean).toFixed(4));

    x.push(lm);
    x.push(mean);
    x.push(now);
    x.sort();

    var y = standard(x, mean, std);
    var content = {
        labels: x,
        datasets: [
            {
                label: kind.textContent + code.value,
                fill: false,
                data: y,
                borderWidth: 3,
                borderColor: 'rgb(100,149,237, 0.5)',
                backgroundColor: [
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                    'rgba(128, 128, 128, 0.2)',
                ],
            }]
    };

    for (var i = 0; i < x.length; i++) {
        if (x[i] == mean) {
            content.datasets[0].backgroundColor[i] = 'rgba(255, 255, 0, 1)';
        }

        if (x[i] == now) {
            content.datasets[0].backgroundColor[i] = 'rgba(0, 255, 0, 1)';
        }
        if (x[i] == lm) {
            content.datasets[0].backgroundColor[i] = 'rgba(255, 0, 0, 1)';
        }
    }

    var canv = document.getElementsByTagName("canvas")[0].getContext("2d");
    var myLineChart = new Chart(canv, {
        type: "line",
        data: content
    });

    document.getElementsByTagName("a")[0].textContent = "行情点评";
    document.getElementById("introduce").style.display = "block";

    var url = window.location.href + "/list/" + kind.value + "_" + code.value;
    request("GET", url, "", table);
}

function table(data) {
    var s = "<tr>\
    <td>date</td>\
    <td>price</td>\
    <td>ratio%</td>\
    </tr>";

    document.getElementsByTagName("tbody")[0].innerHTML = "";
    var tableHTML = document.getElementsByTagName("tbody")[0].innerHTML;

    for (var i = 0; i < data.length; i++) {
        var t = s.replace("date", data[i].date);
        t = t.replace("price", data[i].price);
        t = t.replace("ratio", data[i].ratio);
        tableHTML += t;
    }

    document.getElementsByTagName("tbody")[0].innerHTML = tableHTML;
}
