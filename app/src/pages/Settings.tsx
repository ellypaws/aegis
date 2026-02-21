import { useState, useEffect, useRef } from "react";
import { RotateCcw } from "lucide-react";
import { UI } from "../constants";
import { cn } from "../lib/utils";
import { useSettings, applyTheme, defaultSettings } from "../contexts/SettingsContext";
import type { Theme } from "../types";

import { Patterns } from "../components/Patterns";

function RevertButton({ isModified, onRevert }: { isModified: boolean, onRevert: () => void }) {
    if (!isModified) return null;
    return (
        <button
            onClick={onRevert}
            title="Revert to default"
            className="ml-2 inline-block text-zinc-400 hover:text-red-500 transition-colors align-text-bottom"
        >
            <RotateCcw size={14} />
        </button>
    );
}

export function SettingsModal({ open, onClose }: { open: boolean; onClose: () => void }) {
    const { settings, updateSettings, loading } = useSettings();
    const [localTheme, setLocalTheme] = useState<Theme | undefined>(settings.theme);

    const [localHero, setLocalHero] = useState({
        hero_title: settings.hero_title || "",
        hero_subtitle: settings.hero_subtitle || "",
        hero_description: settings.hero_description || "",
    });

    const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

    useEffect(() => {
        setLocalTheme(settings.theme);
        setLocalHero({
            hero_title: settings.hero_title || "",
            hero_subtitle: settings.hero_subtitle || "",
            hero_description: settings.hero_description || "",
        });
    }, [settings, open]);

    useEffect(() => {
        if (open) {
            document.body.style.overflow = "hidden";
        } else {
            document.body.style.overflow = "unset";
        }
        return () => {
            document.body.style.overflow = "unset";
        };
    }, [open]);

    if (!open) return null;
    if (loading) return null; // Or a spinner in modal

    const handleHeroChange = (key: keyof typeof localHero, value: string) => {
        const newHero = { ...localHero, [key]: value };
        setLocalHero(newHero);

        if (debounceRef.current) clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            updateSettings({ ...settings, ...newHero });
        }, 500);
    };

    const handleThemeChange = (key: keyof Theme, value: string | number) => {
        if (!localTheme) return;
        const newTheme = { ...localTheme, [key]: value };
        setLocalTheme(newTheme);
        applyTheme(newTheme); // Apply live preview
    };

    const saveTheme = () => {
        if (localTheme) {
            updateSettings({ ...settings, theme: localTheme });
            onClose();
        }
    };

    return (
        <div
            onClick={onClose}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4 animate-in fade-in duration-200"
        >
            <div
                onClick={(e) => e.stopPropagation()}
                className={cn(UI.card, "w-full max-w-4xl max-h-[90vh] overflow-y-auto p-6 md:p-8 relative")}
            >
                <div className="absolute inset-0 overflow-hidden rounded-[20px] pointer-events-none">
                    <Patterns.Polka color="rgba(253, 224, 71, 0.15)" />
                    <div className="absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white shadow-lg bg-yellow-400" />
                    <div className="absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white shadow-lg bg-green-400" />
                </div>

                <div className="relative z-10">
                    <button
                        onClick={onClose}
                        className="absolute top-0 right-0 p-2 hover:bg-black/10 rounded-full transition-colors"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </button>

                    <div className="space-y-8">
                        {/* Header */}
                        <div className="flex items-center justify-between pr-8">
                            <div className={cn(
                                "inline-block rounded-xl border-4 bg-white px-4 py-2 rotate-[-2deg]",
                                "border-yellow-400 shadow-[4px_4px_0px_rgba(250,204,21,1)]"
                            )}>
                                <div className="text-2xl font-black uppercase tracking-tight text-yellow-600">
                                    Settings
                                </div>
                            </div>
                        </div>

                        {/* Hero Settings */}
                        <div className={cn(UI.card, "p-6")}>
                            <div className="absolute -top-4 -left-4 -rotate-2 rounded-xl border-4 border-black bg-white px-4 py-1 font-black shadow-[4px_4px_0px_rgba(0,0,0,0.5)] dark:border-white dark:bg-black dark:text-white">
                                HERO TEXT
                            </div>
                            <div className="mt-4 grid gap-4">
                                <div>
                                    <div className="flex items-center justify-between mb-1">
                                        <label className={UI.label}>Hero Title</label>
                                        <RevertButton isModified={localHero.hero_title !== defaultSettings.hero_title} onRevert={() => handleHeroChange("hero_title", defaultSettings.hero_title)} />
                                    </div>
                                    <input
                                        className={UI.input}
                                        value={localHero.hero_title}
                                        onChange={(e) => handleHeroChange("hero_title", e.target.value)}
                                    />
                                </div>
                                <div>
                                    <div className="flex items-center justify-between mb-1">
                                        <label className={UI.label}>Hero Subtitle</label>
                                        <RevertButton isModified={localHero.hero_subtitle !== defaultSettings.hero_subtitle} onRevert={() => handleHeroChange("hero_subtitle", defaultSettings.hero_subtitle)} />
                                    </div>
                                    <input
                                        className={UI.input}
                                        value={localHero.hero_subtitle}
                                        onChange={(e) => handleHeroChange("hero_subtitle", e.target.value)}
                                    />
                                </div>
                                <div>
                                    <div className="flex items-center justify-between mb-1">
                                        <label className={UI.label}>Hero Description</label>
                                        <RevertButton isModified={localHero.hero_description !== defaultSettings.hero_description} onRevert={() => handleHeroChange("hero_description", defaultSettings.hero_description)} />
                                    </div>
                                    <textarea
                                        className={UI.input}
                                        value={localHero.hero_description}
                                        onChange={(e) => handleHeroChange("hero_description", e.target.value)}
                                    />
                                </div>
                            </div>
                        </div>

                        {/* Theme Settings */}
                        {localTheme && (
                            <div className={cn(UI.card, "p-6 mt-12")}>
                                <div className="absolute -top-4 -right-4 rotate-2 rounded-xl border-4 border-black bg-white px-4 py-1 font-black shadow-[4px_4px_0px_rgba(0,0,0,0.5)] dark:border-white dark:bg-black dark:text-white">
                                    THEME POP
                                </div>

                                <div className="mt-4 grid gap-8 md:grid-cols-2">
                                    {/* Universal */}
                                    <div className="space-y-4">
                                        <h3 className={UI.sectionTitle}>Universal</h3>
                                        <div>
                                            <div className="flex items-center justify-between mb-1">
                                                <label className={UI.label}>Border Radius: {localTheme.border_radius}</label>
                                                <RevertButton isModified={localTheme.border_radius !== defaultSettings.theme?.border_radius} onRevert={() => handleThemeChange("border_radius", defaultSettings.theme!.border_radius)} />
                                            </div>
                                            <input
                                                type="range"
                                                min="0"
                                                max="3"
                                                step="0.1"
                                                className="w-full accent-green-500 cursor-pointer"
                                                value={parseFloat(localTheme.border_radius)}
                                                onChange={(e) => handleThemeChange("border_radius", `${e.target.value}rem`)}
                                            />
                                        </div>
                                        <div>
                                            <div className="flex items-center justify-between mb-1">
                                                <label className={UI.label}>Border Size: {localTheme.border_size}</label>
                                                <RevertButton isModified={localTheme.border_size !== defaultSettings.theme?.border_size} onRevert={() => handleThemeChange("border_size", defaultSettings.theme!.border_size)} />
                                            </div>
                                            <input
                                                type="range"
                                                min="0"
                                                max="10"
                                                step="1"
                                                className="w-full accent-green-500 cursor-pointer"
                                                value={parseInt(localTheme.border_size)}
                                                onChange={(e) => handleThemeChange("border_size", `${e.target.value}px`)}
                                            />
                                        </div>
                                    </div>

                                    {/* Light Theme */}
                                    <div className="space-y-4">
                                        <h3 className={UI.sectionTitle}>Light Mode</h3>
                                        <div className="grid grid-cols-2 gap-4">
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Page Bg</label>
                                                    <RevertButton isModified={localTheme.page_bg_light !== defaultSettings.theme?.page_bg_light} onRevert={() => handleThemeChange("page_bg_light", defaultSettings.theme!.page_bg_light)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.page_bg_light} onChange={(e) => handleThemeChange("page_bg_light", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.page_bg_light} onChange={(e) => handleThemeChange("page_bg_light", e.target.value)} />
                                                </div>
                                            </div>
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Card Bg</label>
                                                    <RevertButton isModified={localTheme.card_bg_light !== defaultSettings.theme?.card_bg_light} onRevert={() => handleThemeChange("card_bg_light", defaultSettings.theme!.card_bg_light)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.card_bg_light} onChange={(e) => handleThemeChange("card_bg_light", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.card_bg_light} onChange={(e) => handleThemeChange("card_bg_light", e.target.value)} />
                                                </div>
                                            </div>
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Border Color</label>
                                                    <RevertButton isModified={localTheme.border_color_light !== defaultSettings.theme?.border_color_light} onRevert={() => handleThemeChange("border_color_light", defaultSettings.theme!.border_color_light)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.border_color_light} onChange={(e) => handleThemeChange("border_color_light", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.border_color_light} onChange={(e) => handleThemeChange("border_color_light", e.target.value)} />
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Dark Theme */}
                                    <div className="space-y-4">
                                        <h3 className={UI.sectionTitle}>Dark Mode</h3>
                                        <div className="grid grid-cols-2 gap-4">
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Page Bg</label>
                                                    <RevertButton isModified={localTheme.page_bg_dark !== defaultSettings.theme?.page_bg_dark} onRevert={() => handleThemeChange("page_bg_dark", defaultSettings.theme!.page_bg_dark)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.page_bg_dark} onChange={(e) => handleThemeChange("page_bg_dark", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.page_bg_dark} onChange={(e) => handleThemeChange("page_bg_dark", e.target.value)} />
                                                </div>
                                            </div>
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Card Bg</label>
                                                    <RevertButton isModified={localTheme.card_bg_dark !== defaultSettings.theme?.card_bg_dark} onRevert={() => handleThemeChange("card_bg_dark", defaultSettings.theme!.card_bg_dark)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.card_bg_dark} onChange={(e) => handleThemeChange("card_bg_dark", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.card_bg_dark} onChange={(e) => handleThemeChange("card_bg_dark", e.target.value)} />
                                                </div>
                                            </div>
                                            <div>
                                                <div className="flex items-center justify-between mb-1">
                                                    <label className={UI.label}>Border Color</label>
                                                    <RevertButton isModified={localTheme.border_color_dark !== defaultSettings.theme?.border_color_dark} onRevert={() => handleThemeChange("border_color_dark", defaultSettings.theme!.border_color_dark)} />
                                                </div>
                                                <div className="flex gap-2">
                                                    <input type="color" value={localTheme.border_color_dark} onChange={(e) => handleThemeChange("border_color_dark", e.target.value)} className="h-10 w-10 cursor-pointer rounded border-2 border-zinc-300" />
                                                    <input className={UI.input} value={localTheme.border_color_dark} onChange={(e) => handleThemeChange("border_color_dark", e.target.value)} />
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div className="mt-8 flex justify-end gap-4">
                                    <button onClick={onClose} className={cn(UI.button, UI.btnYellow)}>
                                        Cancel
                                    </button>
                                    <button onClick={saveTheme} className={cn(UI.button, UI.btnGreen)}>
                                        Save Theme
                                    </button>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
