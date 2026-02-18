import { cn, formatBytes } from "../lib/utils";
import { Download } from "lucide-react";

export type DownloadFile = {
    url: string;
    name: string;
    size?: number;
    mime?: string;
};

export function DownloadButton({ files, enabled }: { files: DownloadFile[]; enabled: boolean }) {
    if (!files || files.length === 0) return null;

    return (
        <div className="space-y-3">
            {files.map((file, i) => (
                <a
                    key={i}
                    href={enabled ? file.url : undefined}
                    download={enabled ? file.name : undefined}
                    onClick={(e) => {
                        if (!enabled) e.preventDefault();
                    }}
                    className={cn(
                        "group flex items-center justify-between gap-3 rounded-2xl border-2 px-3 py-2 text-left bg-white dark:bg-zinc-900 transition-all",
                        enabled
                            ? "border-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 shadow-[4px_4px_0px_rgba(96,165,250,1)] dark:shadow-[4px_4px_0px_rgba(59,130,246,0.5)] active:translate-x-[1px] active:translate-y-[1px] active:shadow-none"
                            : "border-zinc-200 dark:border-zinc-800 text-zinc-400 cursor-not-allowed opacity-60 shadow-[3px_3px_0px_rgba(0,0,0,0.10)]"
                    )}
                    title={enabled ? `Download ${file.name}` : "No access"}
                >
                    <div className="min-w-0">
                        <div className={cn("truncate text-xs font-black uppercase tracking-wide", enabled ? "text-blue-700 dark:text-blue-400" : "text-zinc-400 dark:text-zinc-600")}>{file.name}</div>
                        <div className={cn("mt-0.5 text-xs font-bold", enabled ? "text-blue-300 dark:text-blue-300/60" : "text-zinc-300 dark:text-zinc-700")}>
                            {file.size ? formatBytes(file.size) : "Unknown size"} · {file.mime?.replace("application/octet-stream", "image/png") || "image/png"}
                        </div>
                    </div>
                    <div className={cn("shrink-0", enabled ? "text-blue-600 dark:text-blue-400" : "text-zinc-300 dark:text-zinc-700")}>
                        <Download className="w-5 h-5" />
                    </div>
                </a>
            ))}
        </div>
    );
}
