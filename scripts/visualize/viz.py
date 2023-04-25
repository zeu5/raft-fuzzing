import json
import os
from flask import Flask
from flask import jsonify, render_template
from analyze_graph import analyze

app = Flask(__name__)
graph_path = os.environ["GRAPH_PATH"] if "GRAPH_PATH" in os.environ else "../../traces/"

@app.route("/")
def index():
    return render_template("graph.html")

@app.route("/graphs")
def graphs():
    result = []
    for file in os.listdir(graph_path):
        if "visit_graph_" in file:
            name = file.strip("visit_graph_").strip(".json")
            result.append(name)
    return jsonify({
        "graphs": result
    })

@app.route("/graph/<name>")
def graph(name="random"):
    graph = {}
    with open(os.path.join(graph_path,"visit_graph_"+name+".json")) as f:
        graph = analyze(json.load(f))
    return jsonify(graph)