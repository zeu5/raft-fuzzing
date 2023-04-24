import json
import sys

def analyze(graph):
    start_states = set()
    for n in graph["Nodes"].keys():
        node = graph["Nodes"][n]
        if "Prev" not in node or len(node["Prev"].keys()) == 0:
            start_states.add(n)
            
    nodes = graph["Nodes"]
    q = list(start_states)

    while len(q) != 0:
        cur = q.pop(0)
        depths = set()
        if "Depth" in nodes[cur]:
            continue
        
        if "Prev" in nodes[cur]:
            for p in nodes[cur]["Prev"].keys():
                if "Depth" in nodes[p]:
                    depths.add(nodes[p]["Depth"]+1)
            if len(depths) != 0:
                min_depth = min(list(depths))
                nodes[cur]["Depth"] = min_depth
        else:
            nodes[cur]["Depth"] = 0
    
        if "Next" in nodes[cur]:
            for next in nodes[cur]["Next"].keys():
                q.append(next)

    new_graph = {
        "Nodes": nodes,
        "StartStates": list(start_states)
    }
    return new_graph

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: python analyze_graph.py <graph_file>")
        sys.exit(1)
    
    file_path = sys.argv[1]
    try:
        graph = json.load(open(file_path))
    except Exception as e:
        print("error reading graph json: ", str(e))
        sys.exit(1)
    
    new_graph = analyze(graph)
    with open(file_path, "w") as f:
        json.dump(new_graph, f)
