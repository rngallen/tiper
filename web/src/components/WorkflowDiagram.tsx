"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, RotateCcw, UserRound, X } from "lucide-react";
import { useAppearance } from "@/lib/appearance";
import "./workflow-diagram.css";
import {
  Background,
  Controls,
  MarkerType,
  Position,
  ReactFlow,
  Handle,
  applyNodeChanges,
  type Edge,
  type Node,
  type NodeChange,
  type NodeProps,
} from "@xyflow/react";

interface DiagramNode {
  id: string;
  name: string;
  status?: string;
  sequence?: number;
}

interface DiagramEdge {
  from: string;
  to: string;
  actType: string;
  name?: string;
}

type NodeTone = "default" | "current" | "done" | "approved" | "reject";

type StepNodeData = {
  label: string;
  tone: NodeTone;
  status?: string;
  kind: "task" | "approved" | "rejected";
};

const STEP_X = 230;
const CHAIN_Y = 88;
const REJECT_Y = 280;

function isApprovedNode(node: DiagramNode): boolean {
  return node.status === "completed" || /^approved$/i.test(node.name);
}

function isRejectedNode(node: DiagramNode): boolean {
  return node.status === "rejected" || /^rejected$/i.test(node.name);
}

function toneForNode(
  node: DiagramNode,
  currentName?: string,
  visitedNames?: Set<string>,
): NodeTone {
  if (isRejectedNode(node)) return "reject";
  if (isApprovedNode(node)) return "approved";
  if (currentName && node.name === currentName) return "current";
  if (visitedNames?.has(node.name)) return "done";
  return "default";
}

const toneClass: Record<NodeTone, string> = {
  default:
    "border-slate-300 bg-white text-slate-800 shadow-sm dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100",
  current:
    "border-brand-900 bg-brand-800 text-white shadow-md ring-2 ring-amber-400 dark:border-amber-400 dark:bg-brand-800 dark:text-white dark:ring-amber-400",
  done: "border-emerald-400 bg-emerald-50 text-emerald-950 shadow-sm dark:border-emerald-600 dark:bg-emerald-950 dark:text-emerald-100",
  approved:
    "border-emerald-600 bg-emerald-600 text-white shadow-md dark:border-emerald-400 dark:bg-emerald-600",
  reject:
    "border-rose-600 bg-rose-600 text-white shadow-md dark:border-rose-400 dark:bg-rose-700",
};

function StepNode({ data }: NodeProps<Node<StepNodeData>>) {
  const terminal = data.kind !== "task";
  return (
    <div
      className={`wf-step group relative min-w-[9.5rem] max-w-[12rem] cursor-grab rounded-xl border px-3.5 py-2.5 text-center active:cursor-grabbing ${
        terminal ? "rounded-full px-5" : ""
      } ${toneClass[data.tone]}`}
      title={data.status ? `${data.label} (${data.status})` : data.label}
    >
      <Handle type="target" position={Position.Left} id="left" className="wf-handle" />
      <Handle type="target" position={Position.Top} id="top" className="wf-handle" />
      <div className="flex items-center justify-center gap-1.5">
        {data.kind === "approved" ? (
          <Check size={15} strokeWidth={2.5} aria-hidden />
        ) : data.kind === "rejected" ? (
          <X size={15} strokeWidth={2.5} aria-hidden />
        ) : (
          <UserRound size={14} strokeWidth={2} className="shrink-0 opacity-70" aria-hidden />
        )}
        <span className="text-[13px] font-semibold leading-snug">{data.label}</span>
      </div>
      {data.tone === "current" ? (
        <div className="mt-1 text-[10px] font-bold uppercase tracking-wide text-amber-300">
          Current
        </div>
      ) : null}
      <Handle type="source" position={Position.Right} id="right" className="wf-handle" />
      <Handle type="source" position={Position.Bottom} id="bottom" className="wf-handle" />
    </div>
  );
}

const nodeTypes = { step: StepNode };

function agreePathIds(nodes: DiagramNode[], edges: DiagramEdge[]): string[] {
  const sorted = [...nodes].sort(
    (a, b) =>
      (a.sequence ?? 0) - (b.sequence ?? 0) || a.name.localeCompare(b.name),
  );
  const agree = edges.filter((e) => e.actType === "agree");
  if (agree.length === 0) {
    return sorted.filter((n) => !isRejectedNode(n)).map((n) => n.id);
  }
  const outs = new Map<string, string>();
  for (const e of agree) outs.set(e.from, e.to);
  const targets = new Set(agree.map((e) => e.to));
  let start: string | undefined =
    sorted.find((n) => outs.has(n.id) && !targets.has(n.id))?.id ||
    sorted[0]?.id;
  const path: string[] = [];
  const seen = new Set<string>();
  while (start && !seen.has(start)) {
    path.push(start);
    seen.add(start);
    start = outs.get(start);
  }
  return path;
}

function layoutKey(nodes: DiagramNode[]): string {
  return `dfms.wf.layout:${[...nodes.map((n) => n.id)].sort().join(".")}`;
}

function readLayout(key: string): Record<string, { x: number; y: number }> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, { x: number; y: number }>;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function writeLayout(key: string, nodes: Node<StepNodeData>[]) {
  const pos: Record<string, { x: number; y: number }> = {};
  for (const n of nodes) pos[n.id] = n.position;
  try {
    localStorage.setItem(key, JSON.stringify(pos));
  } catch {
    /* quota / private mode */
  }
}

function buildFlow(
  nodes: DiagramNode[],
  edges: DiagramEdge[],
  currentNodeName?: string,
  visitedNames?: Set<string>,
  saved?: Record<string, { x: number; y: number }>,
): { flowNodes: Node<StepNodeData>[]; flowEdges: Edge[] } {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const path = agreePathIds(nodes, edges);
  const pathIndex = new Map(path.map((id, i) => [id, i]));
  const rejectTargets = new Set(
    edges.filter((e) => e.actType === "reject").map((e) => e.to),
  );

  const flowNodes: Node<StepNodeData>[] = [];
  const placed = new Set<string>();

  path.forEach((id, i) => {
    const n = byId.get(id);
    if (!n) return;
    placed.add(id);
    const kind: StepNodeData["kind"] = isApprovedNode(n)
      ? "approved"
      : isRejectedNode(n)
        ? "rejected"
        : "task";
    flowNodes.push({
      id,
      type: "step",
      position: saved?.[id] ?? { x: i * STEP_X, y: CHAIN_Y },
      data: {
        label: n.name,
        tone: toneForNode(n, currentNodeName, visitedNames),
        status: n.status,
        kind,
      },
      draggable: true,
    });
  });

  let extra = 0;
  for (const n of nodes) {
    if (placed.has(n.id)) continue;
    const reject = isRejectedNode(n) || rejectTargets.has(n.id);
    const fallback = reject
      ? { x: Math.max(0, (path.length - 1) * STEP_X), y: REJECT_Y }
      : { x: extra * STEP_X, y: REJECT_Y };
    const kind: StepNodeData["kind"] = isApprovedNode(n)
      ? "approved"
      : isRejectedNode(n)
        ? "rejected"
        : "task";
    flowNodes.push({
      id: n.id,
      type: "step",
      position: saved?.[n.id] ?? fallback,
      data: {
        label: n.name,
        tone: toneForNode(n, currentNodeName, visitedNames),
        status: n.status,
        kind,
      },
      draggable: true,
    });
    placed.add(n.id);
    extra++;
  }

  const currentId = currentNodeName
    ? [...byId.values()].find((n) => n.name === currentNodeName)?.id
    : undefined;

  const flowEdges: Edge[] = edges.map((e, i) => {
    const isReject = e.actType === "reject";
    const isAgree = e.actType === "agree";
    const fromOnPath = pathIndex.has(e.from);
    const toOnPath = pathIndex.has(e.to);
    const dropDown = isReject || (fromOnPath && !toOnPath);
    const stroke = isReject ? "#e11d48" : isAgree ? "#059669" : "#64748b";

    return {
      id: `${e.from}-${e.to}-${e.actType}-${i}`,
      source: e.from,
      target: e.to,
      type: "smoothstep",
      sourceHandle: dropDown ? "bottom" : "right",
      targetHandle: dropDown ? "top" : "left",
      animated: !!currentId && e.to === currentId && isAgree,
      style: {
        stroke,
        strokeWidth: isReject ? 2 : 2.5,
        strokeDasharray: isReject ? "7 5" : undefined,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: stroke,
        width: 16,
        height: 16,
      },
      pathOptions: { borderRadius: 16, offset: 18 },
    };
  });

  return { flowNodes, flowEdges };
}

/** Interactive approval-flow graph (React Flow) from process nodes + edges. */
export function WorkflowDiagram({
  nodes,
  edges,
  currentNodeName,
  visitedNodeNames,
  title = "Approval flow",
  heightClass = "h-[28rem]",
}: {
  nodes: DiagramNode[];
  edges: DiagramEdge[];
  currentNodeName?: string;
  visitedNodeNames?: string[];
  title?: string;
  heightClass?: string;
}) {
  const { colorScheme } = useAppearance();
  const dark = colorScheme === "dark";
  const visited = useMemo(
    () => new Set(visitedNodeNames || []),
    [visitedNodeNames],
  );
  const storeKey = useMemo(() => layoutKey(nodes), [nodes]);

  const graphKey = useMemo(() => {
    const n = nodes
      .map((x) => `${x.id}:${x.name}:${x.sequence ?? 0}:${x.status ?? ""}`)
      .join("|");
    const e = edges
      .map((x) => `${x.from}>${x.to}:${x.actType}:${x.name ?? ""}`)
      .join("|");
    return `${n}::${e}`;
  }, [nodes, edges]);

  const built = useMemo(() => {
    const saved = readLayout(storeKey);
    return buildFlow(nodes, edges, currentNodeName, visited, saved);
  }, [nodes, edges, currentNodeName, visited, storeKey, graphKey]);

  const [rfNodes, setRfNodes] = useState<Node<StepNodeData>[]>(built.flowNodes);
  const [rfEdges, setRfEdges] = useState<Edge[]>(built.flowEdges);

  useEffect(() => {
    setRfNodes(built.flowNodes);
    setRfEdges(built.flowEdges);
  }, [built]);

  const onNodesChange = useCallback((changes: NodeChange<Node<StepNodeData>>[]) => {
    setRfNodes((curr) => applyNodeChanges(changes, curr));
  }, []);

  const persistLayout = useCallback(() => {
    setRfNodes((curr) => {
      writeLayout(storeKey, curr);
      return curr;
    });
  }, [storeKey]);

  const resetLayout = useCallback(() => {
    try {
      localStorage.removeItem(storeKey);
    } catch {
      /* ignore */
    }
    const fresh = buildFlow(nodes, edges, currentNodeName, visited);
    setRfNodes(fresh.flowNodes);
    setRfEdges(fresh.flowEdges);
  }, [storeKey, nodes, edges, currentNodeName, visited]);

  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  if (nodes.length === 0) {
    return (
      <p className="text-sm text-slate-400">No workflow steps defined.</p>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/80 dark:border-[var(--pp-border)] dark:bg-[#121820]">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-3 py-2 dark:border-[var(--pp-border)]">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-[var(--pp-text)]">{title}</h3>
          <p className="text-[11px] text-slate-500">
            Drag a step to move it so approve and reject lines stay clear. Pan the canvas; scroll to zoom.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-xs text-slate-600 dark:text-[var(--pp-muted)]">
          <Legend swatch="border-amber-400 bg-brand-800" label="Current" light />
          <Legend swatch="border-emerald-400 bg-emerald-50 dark:bg-emerald-950" label="Done" />
          <Legend swatch="border-emerald-600 bg-emerald-600" label="Approved" light />
          <Legend swatch="border-rose-600 bg-rose-600" label="Rejected" light />
          <Legend swatch="border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-800" label="Upcoming" />
          <Legend line="bg-emerald-600" label="Approve" />
          <Legend line="bg-rose-600 dashed" label="Reject" />
          <button
            type="button"
            onClick={resetLayout}
            className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2 py-1 text-[11px] font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          >
            <RotateCcw size={12} aria-hidden />
            Reset layout
          </button>
        </div>
      </div>

      <div className={`${heightClass} w-full`}>
        {mounted ? (
          <ReactFlow
            nodes={rfNodes}
            edges={rfEdges}
            onNodesChange={onNodesChange}
            onNodeDragStop={persistLayout}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.22, minZoom: 0.55, maxZoom: 1.2 }}
            minZoom={0.25}
            maxZoom={1.75}
            panOnDrag
            panOnScroll={false}
            zoomOnScroll
            zoomOnPinch
            zoomOnDoubleClick
            preventScrolling
            nodesDraggable
            nodesConnectable={false}
            elementsSelectable
            selectNodesOnDrag={false}
            proOptions={{ hideAttribution: true }}
            defaultEdgeOptions={{ type: "smoothstep" }}
          >
            <Background gap={18} color={dark ? "#3d4d63" : "#dbe3ec"} />
            <Controls showInteractive={false} />
          </ReactFlow>
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-slate-400">
            Loading diagram…
          </div>
        )}
      </div>
    </div>
  );
}

function Legend({
  swatch,
  line,
  label,
  light,
}: {
  swatch?: string;
  line?: string;
  label: string;
  light?: boolean;
}) {
  return (
    <span className="inline-flex items-center gap-1.5">
      {line ? (
        <span
          className={`inline-block h-0.5 w-5 rounded ${
            line.includes("dashed") ? "bg-transparent" : line.replace(" dashed", "")
          }`}
          style={
            line.includes("dashed")
              ? {
                  backgroundImage: line.includes("rose")
                    ? "repeating-linear-gradient(90deg,#e11d48 0 4px,transparent 4px 7px)"
                    : "repeating-linear-gradient(90deg,#059669 0 4px,transparent 4px 7px)",
                  height: 2,
                }
              : undefined
          }
        />
      ) : (
        <span
          className={`h-2.5 w-2.5 rounded-sm border ${swatch || ""} ${light ? "ring-1 ring-black/10" : ""}`}
        />
      )}
      {label}
    </span>
  );
}
