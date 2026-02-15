import { cn } from "../lib/utils";
import type { DiscordUser } from "../types";
import { Modal } from "./Modal";

export function LoginModal({
    open,
    onClose,
}: {
    open: boolean;
    onClose: () => void;
    onLogin: (u: DiscordUser) => void;
}) {
    return (
        <Modal open={open} title="Discord Login" onClose={onClose}>
            <div className="text-sm font-bold text-zinc-700">Connect with Discord to access restricted content.</div>

            <div className="mt-4 grid gap-3">
                <button
                    type="button"
                    onClick={() => {
                        window.location.href = "/login";
                    }}
                    className={cn(
                        "w-full rounded-3xl border-4 bg-[#5865F2] p-4 text-left text-white",
                        "hover:bg-[#4752C4] shadow-[6px_6px_0px_rgba(0,0,0,0.16)]",
                        "active:translate-x-[1px] active:translate-y-[1px]"
                    )}
                >
                    <div className="flex items-center justify-center gap-3">
                        <div className="text-base font-black uppercase tracking-wide">Login with Discord</div>
                    </div>
                </button>
            </div>

            <div className="mt-4 text-xs font-bold text-zinc-400">You will be redirected to Discord to authorize.</div>
        </Modal>
    );
}
