$(() => {
    $("#get-graph").click(() => {
        var graph_name = $("#graph").val();
        $.ajax("/graph/" + graph_name).done((graph) => viz_graph(graph));
    });
});

function viz_graph(graph) {

    var maxVisits = 0
    for (var n in graph.Nodes) {
        var node = graph.Nodes[n];
        if (node.Visits > maxVisits) {
            maxVisits = node.Visits;
        }
    }

    var radius = d3.scaleSqrt().domain([0, maxVisits]).range([0, 10]);

    var svg = d3.select("svg")
    svg.append("g")
        .attr("stroke", "black")
        .selectAll("circle")
        .data(Object.values(graph.Nodes))
        .join("circle")
        .attr("cx", d => d.Sibling * 20 + 10)
        .attr("cy", d => d.Depth * 20)
        .attr("r", d => radius(d.Visits))
        .attr("fill", "white")
        .call(circle => circle.append("title")
            .text(d => ["Visits: " + d.Visits, "State:", d.State].join("\n")));

}