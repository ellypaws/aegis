import React, { createContext, useContext, useEffect, useState } from "react";
import type { Settings } from "../types";

interface SettingsContextType {
    settings: Settings;
    loading: boolean;
    refreshSettings: () => Promise<void>;
    updateSettings: (newSettings: Settings) => Promise<void>;
}

const defaultSettings: Settings = {
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
