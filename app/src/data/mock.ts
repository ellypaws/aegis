import { Guild, DiscordUser } from "../types";

export const MOCK_GUILD: Guild = {
    id: "guild_1",
    name: "Keiau Club",
    roles: [
        { id: "r_free", name: "Free", color: "bg-zinc-600" },
        { id: "r_tier1", name: "Tier 1", color: "bg-sky-600" },
        { id: "r_tier2", name: "Tier 2", color: "bg-emerald-600" },
        { id: "r_vip", name: "VIP", color: "bg-fuchsia-600" },
        { id: "r_author", name: "Author", color: "bg-amber-500" },
    ],
    channels: [
        { id: "c_ann", name: "announcements" },
        { id: "c_art", name: "art-drops" },
        { id: "c_vip", name: "vip-lounge" },
    ],
};

export const MOCK_USERS: DiscordUser[] = [
    {
        id: "u_elly",
        username: "Elly",
        avatarUrl: "",
        roles: [MOCK_GUILD.roles.find((r) => r.id === "r_author")!, MOCK_GUILD.roles.find((r) => r.id === "r_vip")!],
        isAuthor: true,
    },
    {
        id: "u_viewer1",
        username: "Mochi",
        avatarUrl: "",
        roles: [MOCK_GUILD.roles.find((r) => r.id === "r_tier1")!],
    },
    {
        id: "u_viewer2",
        username: "Pebble",
        avatarUrl: "",
        roles: [MOCK_GUILD.roles.find((r) => r.id === "r_free")!],
    },
];

export function getRoleName(roleId: string) {
    return MOCK_GUILD.roles.find((r) => r.id === roleId)?.name ?? roleId;
}
