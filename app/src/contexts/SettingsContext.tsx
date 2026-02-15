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
            // Revert on error? Or just let the user know.
            // For now, we just log.
            await refreshSettings();
        }
    };

    return (
        <SettingsContext.Provider value={{ settings, loading, refreshSettings, updateSettings }}>
            {children}
        </SettingsContext.Provider>
    );
}
