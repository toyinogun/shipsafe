import React from "react";
import { TERMINAL_DEMO } from "../data/mockData";

interface TerminalDemoProps {
  readonly className?: string;
}

export const TerminalDemo: React.FC<TerminalDemoProps> = ({ className = "" }) => {
  return (
    <div
      className={`relative max-w-4xl mx-auto glow-effect rounded-xl bg-surface-darker border border-slate-700/50 shadow-2xl overflow-hidden ${className}`}
    >
      {/* Terminal Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-[#0d131c] border-b border-white/5">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-red-500/80" />
          <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
          <div className="w-3 h-3 rounded-full bg-green-500/80" />
        </div>
        <div className="text-xs font-mono text-slate-500">
          {TERMINAL_DEMO.title}
        </div>
        <div className="w-16" />
      </div>

      {/* Terminal Body */}
      <div className="p-6 font-mono text-sm md:text-base overflow-x-auto">
        <div className="flex flex-col gap-2">
          {/* Command */}
          <div className="flex text-slate-400">
            <span className="text-green-500 mr-2">&gt;</span>
            <span>{TERMINAL_DEMO.command}</span>
          </div>

          {/* Init lines */}
          {TERMINAL_DEMO.initLines.map((line) => (
            <div key={line} className="text-slate-500">
              {line}
            </div>
          ))}

          {/* Code block */}
          <div className="mt-4 p-4 rounded-sm bg-[#111821] border-l-2 border-primary/50 relative">
            <div className="absolute top-2 right-2 text-xs text-slate-500">
              {TERMINAL_DEMO.filename}
            </div>
            <pre className="text-slate-300">
              <code>
                <span className="code-keyword">async function</span>{" "}
                <span className="code-func">processUserData</span>(data) {"{"}
                {"\n"}
                {"  "}
                <span className="code-comment">
                  {"// AI Generated Block Start"}
                </span>
                {"\n"}
                {"  "}
                <span className="code-keyword">const</span> sanitized = data.
                <span className="code-func">replace</span>(
                <span className="code-string">/[^\w\s]/gi</span>,{" "}
                <span className="code-string">{"''"}</span>);{"\n"}
                {"  "}
                <span className="code-keyword">await</span> db.
                <span className="code-func">query</span>(
                <span className="code-string">
                  {"`SELECT * FROM users WHERE id = "}
                  <span className="text-primary font-bold">
                    {"${sanitized}"}
                  </span>
                  {"`"}
                </span>
                );{"\n"}
                {"  "}
                <span className="code-comment">
                  {"// AI Generated Block End"}
                </span>
                {"\n"}
                {"}"}
              </code>
            </pre>
          </div>

          {/* Divider */}
          <div className="mt-4 flex items-center gap-3">
            <div className="h-px bg-slate-700 flex-grow" />
            <span className="text-slate-500 text-xs uppercase tracking-wider">
              {TERMINAL_DEMO.resultLabel}
            </span>
            <div className="h-px bg-slate-700 flex-grow" />
          </div>

          {/* Result */}
          <div className="mt-4 flex flex-col md:flex-row gap-4 items-start md:items-center justify-between bg-surface-dark border border-green-500/20 rounded-lg p-4">
            <div>
              <div className="text-slate-300 font-medium mb-1">
                {TERMINAL_DEMO.resultTitle}
              </div>
              <div className="text-slate-500 text-xs">
                {TERMINAL_DEMO.resultSubtitle}
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="px-3 py-1 rounded-sm bg-green-500/10 border border-green-500/20 text-green-400 text-sm font-bold flex items-center gap-2">
                <span className="material-symbols-outlined text-sm">
                  check_circle
                </span>
                {TERMINAL_DEMO.status}
              </div>
              <div className="text-2xl font-bold text-white">
                {TERMINAL_DEMO.score}
                <span className="text-slate-600 text-base font-normal">
                  /100
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TerminalDemo;
