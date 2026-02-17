import { cn, formatBytes } from "../lib/utils";
// import { FileRef } from "../types";

export type DownloadFile = {
    url: string;
    name: string;
    size?: number;
    mime?: string;
};

export function DownloadButton({ file, enabled }: { file: DownloadFile; enabled: boolean }) {
    return (
        <a
            href={enabled ? file.url : undefined}
            download={enabled ? file.name : undefined}
            onClick={(e) => {
                if (!enabled) e.preventDefault();
            }}
            className={cn(
                "group flex items-center justify-between gap-3 rounded-2xl border-4 px-4 py-3 text-left bg-white",
                enabled
                    ? "border-blue-300 hover:bg-blue-50 shadow-[4px_4px_0px_rgba(96,165,250,1)]"
                    : "border-zinc-200 text-zinc-400 cursor-not-allowed opacity-60 shadow-[3px_3px_0px_rgba(0,0,0,0.10)]",
                "transition active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title={enabled ? `Download ${file.name}` : "No access"}
        >
            <div className="min-w-0">
                <div className={cn("truncate text-sm font-black uppercase tracking-wide", enabled ? "text-blue-700" : "text-zinc-400")}>{file.name}</div>
                <div className={cn("mt-0.5 text-xs font-bold", enabled ? "text-blue-300" : "text-zinc-300")}>
                    {file.size ? formatBytes(file.size) : "Unknown size"} · {file.mime?.replace("application/octet-stream", "image/png") || "image/png"}
                </div>
            </div>
            <div className={cn("shrink-0 text-lg", enabled ? "text-blue-600" : "text-zinc-300")}>⬇</div>
        </a>
    );
}
