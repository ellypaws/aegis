import React, { useState, useEffect } from "react";
import { useSettings } from "../contexts/SettingsContext";
import { UI } from "../constants";
import { cn } from "../lib/utils";
import type { DiscordUser } from "../types";

export function Settings({ user }: { user: DiscordUser | null }) {
    const { settings, loading, updateSettings } = useSettings();
    const [formData, setFormData] = useState(settings);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        setFormData(settings);
    }, [settings]);

    if (!user?.isAdmin) {
        return <div className="p-12 text-center font-bold text-red-500">Access Denied</div>;
    }

    if (loading) {
        return <div className="p-12 text-center font-bold text-zinc-400">Loading settings...</div>;
    }

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const { name, value } = e.target;
        setFormData(prev => ({ ...prev, [name]: value }));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSaving(true);
        await updateSettings(formData);
        setSaving(false);
        alert("Settings saved!");
    };

    return (
        <div className={cn(UI.max, "mt-8")}>
            <div className={cn(UI.card, "p-8 max-w-2xl mx-auto")}>
                <div className="absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white bg-yellow-400 shadow-lg pointer-events-none" />

                <h1 className="text-2xl font-black uppercase tracking-tight text-zinc-900 mb-6 relative z-10">App Settings</h1>

                <form onSubmit={handleSubmit} className="space-y-6 relative z-10">
                    <div className="space-y-1">
                        <label className={UI.label}>Hero Title</label>
                        <input
                            name="hero_title"
                            value={formData.hero_title}
                            onChange={handleChange}
                            className={UI.input}
                            placeholder="e.g. Tiered Vault"
                        />
                        <p className="text-xs text-zinc-400">The main red badge on the home header.</p>
                    </div>

                    <div className="space-y-1">
                        <label className={UI.label}>Hero Subtitle</label>
                        <input
                            name="hero_subtitle"
                            value={formData.hero_subtitle}
                            onChange={handleChange}
                            className={UI.input}
                            placeholder="e.g. How it works"
                        />
                        <p className="text-xs text-zinc-400">The blue badge title.</p>
                    </div>

                    <div className="space-y-1">
                        <label className={UI.label}>Hero Description</label>
                        <textarea
                            name="hero_description"
                            value={formData.hero_description}
                            onChange={handleChange}
                            className={cn(UI.input, "resize-none")}
                            rows={3}
                            placeholder="e.g. Upload full + thumbnail..."
                        />
                        <p className="text-xs text-zinc-400">The description text in the blue box.</p>
                    </div>

                    <div className="pt-4 flex justify-end gap-3">
                        <button
                            type="submit"
                            disabled={saving}
                            className={cn(UI.button, UI.btnYellow, saving && UI.btnDisabled)}
                        >
                            {saving ? "Saving..." : "Save Changes"}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
