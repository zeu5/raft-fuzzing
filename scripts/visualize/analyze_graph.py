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
    edges = dict()

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
                edges[(cur, next)] = [cur,next]
                q.append(next)

    depths = dict()
    for node in graph["Nodes"].values():
        if "Sibling" in node:
            continue
        if node["Depth"] not in depths:
            depths[node["Depth"]] = set()
        depths[node["Depth"]].add(node["Key"])
    
    for d in depths:
        d_nodes = list(depths[d])
        d_nodes.sort()
        for (i,n) in enumerate(d_nodes):
            nodes[str(n)]["Sibling"] = i

    new_graph = {
        "Nodes": nodes,
        "StartStates": list(start_states),
        "Edges": [edges[e] for e in edges],
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
