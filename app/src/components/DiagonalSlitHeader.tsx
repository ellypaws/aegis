import { useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { Patterns } from "./Patterns";
import { useSettings } from "../contexts/SettingsContext";

export function DiagonalSlitHeader({ posts, onClickRandom }: { posts: Post[]; onClickRandom: () => void }) {
    const { settings } = useSettings();

    const tiles = useMemo(() => {
        const src = posts
            .slice()
            .sort(() => Math.random() - 0.5)
            .slice(0, 10)
            .map((p) => p.images?.[0]?.thumbnail || p.images?.[0]?.blobs?.[0]?.data);
        return src;
    }, [posts]);

    return (
        <div className="relative overflow-hidden rounded-3xl border-b-8 border-r-8 border-yellow-500/30 dark:border-yellow-600/30 bg-yellow-300 dark:bg-yellow-700">
            <Patterns.Polka color="rgba(255,0,0,0.10)" />

            <div className="pointer-events-none absolute top-[-20px] left-[-20px] h-32 w-32 rounded-full border-4 border-white dark:border-zinc-300 bg-red-400 shadow-lg" />
            <div className="pointer-events-none absolute bottom-[-10px] right-[-10px] h-24 w-24 rotate-12 border-4 border-white dark:border-zinc-300 bg-blue-400 shadow-lg" />
            <div className="pointer-events-none absolute top-10 right-10 h-12 w-12 rotate-45 rounded-lg border-4 border-white dark:border-zinc-300 bg-green-400 shadow-md" />

            <div className="absolute inset-0">
                <div className="h-full w-full opacity-85">
                    <div className="grid h-full w-full grid-cols-5 gap-1 p-2">
                        {Array.from({ length: 10 }).map((_, i) => {
                            const url = tiles[i % Math.max(1, tiles.length)];
                            return (
                                <div key={i} className="relative overflow-hidden rounded-2xl border-4 border-white/60 dark:border-white/10 bg-white/30 dark:bg-black/10">
                                    {url ? <img src={url} className="h-full w-full object-cover" alt="" draggable={false} /> : null}
                                    <div className="absolute inset-0 bg-gradient-to-t from-yellow-300/70 dark:from-yellow-700/70 to-transparent" />
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>

            <div
                className={cn(
                    "pointer-events-none absolute inset-0",
                    "bg-yellow-300/85 dark:bg-yellow-700/85",
                    "[mask-image:linear-gradient(110deg,transparent_0%,transparent_34%,black_36%,black_62%,transparent_64%,transparent_100%)]",
                    "[mask-size:100%_100%]"
                )}
            />

            <div className="relative z-10 flex items-center justify-between gap-4 px-6 py-6">
                <div className="space-y-2">
                    <div className="inline-block rounded-xl border-4 border-red-400 dark:border-red-600 bg-white dark:bg-zinc-800 px-4 py-2 rotate-[-2deg] shadow-[4px_4px_0px_rgba(248,113,113,1)] dark:shadow-[4px_4px_0px_rgba(220,38,38,0.5)]">
                        <div className="text-xl font-black uppercase tracking-tight text-red-500 dark:text-red-300">{settings.hero_title}</div>
                    </div>
                    <div className="max-w-xl rounded-2xl border-4 border-blue-400 dark:border-blue-600 bg-white dark:bg-zinc-800 px-4 py-3 shadow-[4px_4px_0px_rgba(96,165,250,1)] dark:shadow-[4px_4px_0px_rgba(59,130,246,0.5)]">
                        <div className="text-xs font-black uppercase tracking-wide text-blue-300 dark:text-blue-400">{settings.hero_subtitle}</div>
                        <div className="text-sm font-bold text-blue-700 dark:text-blue-300">{settings.hero_description}</div>
                    </div>
                </div>

                <button type="button" onClick={onClickRandom} className={cn(UI.button, UI.btnGreen)}>
                    Random
                </button>
            </div>
        </div>
    );
}
