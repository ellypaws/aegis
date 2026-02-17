export const UI = {
    // Dynamic page bg with themed background images
    page: "min-h-screen font-sans transition-all duration-300 bg-[var(--theme-page-bg-light)] dark:bg-[var(--theme-page-bg-dark)] text-zinc-900 dark:text-zinc-100 bg-[image:var(--theme-page-bg-image-light)] dark:bg-[image:var(--theme-page-bg-image-dark)] bg-cover bg-center bg-fixed",
    max: "mx-auto max-w-7xl px-4 py-6",

    // Cards - using dynamic border radius/size/colors
    card:
        "relative rounded-[var(--theme-radius)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-[var(--theme-border-light)] dark:border-[var(--theme-border-dark)] shadow-[6px_6px_0px_rgba(0,0,0,0.18)] dark:shadow-[6px_6px_0px_rgba(255,255,0,0.1)] transition-colors duration-300",
    cardBlue:
        "relative rounded-[var(--theme-radius)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-blue-300 dark:border-blue-700 shadow-[6px_6px_0px_rgba(96,165,250,1)] dark:shadow-[6px_6px_0px_rgba(59,130,246,0.5)] transition-colors duration-300",
    cardRed:
        "relative rounded-[var(--theme-radius)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-red-300 dark:border-red-700 shadow-[6px_6px_0px_rgba(248,113,113,1)] dark:shadow-[6px_6px_0px_rgba(239,68,68,0.5)] transition-colors duration-300",
    cardGreen:
        "relative rounded-[var(--theme-radius)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-green-300 dark:border-green-700 shadow-[6px_6px_0px_rgba(74,222,128,1)] dark:shadow-[6px_6px_0px_rgba(34,197,94,0.5)] transition-colors duration-300",

    soft:
        "rounded-[calc(var(--theme-radius)-0.5rem)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-zinc-200 dark:border-zinc-700 shadow-[4px_4px_0px_rgba(0,0,0,0.12)] dark:shadow-[4px_4px_0px_rgba(0,0,0,0.5)] transition-colors duration-300",

    pill:
        "inline-flex items-center gap-2 rounded-full border-2 border-zinc-200 dark:border-zinc-700 bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] px-2 py-0.5 text-xs font-black uppercase tracking-wide shadow-[2px_2px_0px_rgba(0,0,0,0.10)] dark:shadow-[2px_2px_0px_rgba(0,0,0,0.5)] transition-colors duration-300",

    button:
        "rounded-[calc(var(--theme-radius)-0.5rem)] border-[length:var(--theme-border-size)] px-4 py-2 text-sm font-black uppercase tracking-wide shadow-[4px_4px_0px_rgba(0,0,0,0.18)] dark:shadow-[4px_4px_0px_rgba(0,0,0,0.5)] transition cursor-pointer hover:scale-105 active:scale-95 active:translate-x-[1px] active:translate-y-[1px] active:shadow-[2px_2px_0px_rgba(0,0,0,0.18)]",
    
    btnBlue: "border-blue-400 bg-blue-200 text-blue-900 dark:border-blue-700 dark:bg-blue-900 dark:text-blue-100 hover:bg-blue-300 dark:hover:bg-blue-800",
    btnRed: "border-red-400 bg-red-200 text-red-900 dark:border-red-700 dark:bg-red-900 dark:text-red-100 hover:bg-red-300 dark:hover:bg-red-800",
    btnGreen: "border-green-400 bg-green-200 text-green-900 dark:border-green-700 dark:bg-green-900 dark:text-green-100 hover:bg-green-300 dark:hover:bg-green-800",
    
    btnYellow: "border-yellow-400 bg-yellow-200 text-yellow-900 dark:border-yellow-700 dark:bg-yellow-900 dark:text-yellow-100 hover:bg-yellow-300 dark:hover:bg-yellow-800",

    btnDisabled: "opacity-45 cursor-not-allowed hover:bg-inherit",

    input:
        "w-full rounded-[var(--theme-radius)] border-[length:var(--theme-border-size)] border-zinc-200 dark:border-zinc-700 bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:border-zinc-300 dark:focus:border-zinc-600 focus:outline-none shadow-[3px_3px_0px_rgba(0,0,0,0.10)] dark:shadow-[3px_3px_0px_rgba(0,0,0,0.5)] transition-colors duration-300",

    label: "text-[11px] font-black uppercase tracking-wide text-zinc-500 dark:text-zinc-400",
    sectionTitle: "text-sm font-black uppercase tracking-wide text-zinc-800 dark:text-zinc-200",
};
