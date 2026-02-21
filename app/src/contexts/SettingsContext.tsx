import React, { createContext, useContext, useEffect, useState } from "react";
import type { Settings, Theme } from "../types";

interface SettingsContextType {
    settings: Settings;
    loading: boolean;
    refreshSettings: () => Promise<void>;
    updateSettings: (newSettings: Settings) => Promise<void>;
}

export const defaultSettings: Settings = {
    hero_title: "Tiered Vault",
    hero_subtitle: "How it works",
    hero_description: "Upload full + thumbnail. Gate access by Discord roles. Browse locked previews.",
    public_access: false,
    theme: {
        border_radius: "1.5rem",
        border_size: "4px",
        primary_color_light: "#fde047",
        secondary_color_light: "#fef08a",
        page_bg_light: "#fef08a",
        page_bg_trans_light: 1.0,
        card_bg_light: "#ffffff",
        card_bg_trans_light: 1.0,
        border_color_light: "#fde047",
        primary_color_dark: "#ca8a04",
        secondary_color_dark: "#a16207",
        page_bg_dark: "#09090b",
        page_bg_trans_dark: 1.0,
        card_bg_dark: "#18181b",
        card_bg_trans_dark: 1.0,
        border_color_dark: "#ca8a04",
    }
};

const SettingsContext = createContext<SettingsContextType>({
    settings: defaultSettings,
    loading: true,
    refreshSettings: async () => { },
    updateSettings: async () => { },
});

export function useSettings() {
    return useContext(SettingsContext);
}

export function applyTheme(t?: Partial<Theme>) {
    if (!t) return;
    const root = document.documentElement;

    if (t.border_radius) root.style.setProperty("--theme-radius", t.border_radius);
    if (t.border_size) root.style.setProperty("--theme-border-size", t.border_size);

    if (t.primary_color_light) root.style.setProperty("--theme-primary-light", t.primary_color_light);
    if (t.secondary_color_light) root.style.setProperty("--theme-secondary-light", t.secondary_color_light);
    if (t.page_bg_light) root.style.setProperty("--theme-page-bg-light", t.page_bg_light);
    if (t.card_bg_light) root.style.setProperty("--theme-card-bg-light", t.card_bg_light);
    if (t.border_color_light) root.style.setProperty("--theme-border-light", t.border_color_light);

    if (t.primary_color_dark) root.style.setProperty("--theme-primary-dark", t.primary_color_dark);
    if (t.secondary_color_dark) root.style.setProperty("--theme-secondary-dark", t.secondary_color_dark);
    if (t.page_bg_dark) root.style.setProperty("--theme-page-bg-dark", t.page_bg_dark);
    if (t.card_bg_dark) root.style.setProperty("--theme-card-bg-dark", t.card_bg_dark);
    if (t.border_color_dark) root.style.setProperty("--theme-border-dark", t.border_color_dark);
}

export function SettingsProvider({ children }: { children: React.ReactNode }) {
    const [settings, setSettings] = useState<Settings>(defaultSettings);
    const [loading, setLoading] = useState(true);

    const refreshSettings = async () => {
        try {
            const res = await fetch("/settings");
            if (res.ok) {
                const data = await res.json();
                setSettings(data);
            }
        } catch (error) {
            console.error("Failed to fetch settings:", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        refreshSettings();
    }, []);

    useEffect(() => {
        applyTheme(settings.theme);
    }, [settings.theme]);

    const updateSettings = async (newSettings: Settings) => {
        // Optimistically update
        setSettings(newSettings);

        try {
            const token = localStorage.getItem("jwt");
            const headers: Record<string, string> = {
                "Content-Type": "application/json",
            };
            if (token) headers["Authorization"] = `Bearer ${token}`;

            const res = await fetch("/settings", {
                method: "POST",
                headers,
                body: JSON.stringify(newSettings),
            });

            if (!res.ok) {
                throw new Error("Failed to update settings");
            }

            const updated = await res.json();
            setSettings(updated);
        } catch (error) {
            console.error("Failed to update settings:", error);
            await refreshSettings();
        }
    };

    return (
        <SettingsContext.Provider value={{ settings, loading, refreshSettings, updateSettings }}>
            {children}
        </SettingsContext.Provider>
    );
}
