import type { Guild, DiscordUser, DiscordRole } from "../types";

export const MOCK_GUILD: Guild = {
    id: "guild_1",
    name: "Keiau Club",
    roles: [
        { roleId: "r_free", name: "Free", color: 0x52525b, managed: false, mentionable: false, hoist: false, position: 0, permissions: 0, icon: "", unicodeEmoji: "", flags: 0 },
        { roleId: "r_tier1", name: "Tier 1", color: 0x0284c7, managed: false, mentionable: false, hoist: false, position: 1, permissions: 0, icon: "", unicodeEmoji: "", flags: 0 },
        { roleId: "r_tier2", name: "Tier 2", color: 0x059669, managed: false, mentionable: false, hoist: false, position: 2, permissions: 0, icon: "", unicodeEmoji: "", flags: 0 },
        { roleId: "r_vip", name: "VIP", color: 0xc026d3, managed: false, mentionable: false, hoist: false, position: 3, permissions: 0, icon: "", unicodeEmoji: "", flags: 0 },
        { roleId: "r_author", name: "Author", color: 0xf59e0b, managed: false, mentionable: false, hoist: false, position: 4, permissions: 0, icon: "", unicodeEmoji: "", flags: 0 },
    ],
    channels: [
        { id: "c_ann", name: "announcements" },
        { id: "c_art", name: "art-drops" },
        { id: "c_vip", name: "vip-lounge" },
    ],
};

export const MOCK_USERS: DiscordUser[] = [
    {
        userId: "u_elly",
        username: "Elly",
        globalName: "Elly",
        discriminator: "0000",
        avatar: "",
        banner: "",
        accentColor: 0,
        bot: false,
        system: false,
        publicFlags: 0,
        roles: [MOCK_GUILD.roles.find((r) => r.roleId === "r_author")!, MOCK_GUILD.roles.find((r) => r.roleId === "r_vip")!],
    },
    {
        userId: "u_viewer1",
        username: "Mochi",
        globalName: "Mochi",
        discriminator: "0000",
        avatar: "",
        banner: "",
        accentColor: 0,
        bot: false,
        system: false,
        publicFlags: 0,
        roles: [MOCK_GUILD.roles.find((r) => r.roleId === "r_tier1")!],
    },
    {
        userId: "u_viewer2",
        username: "Pebble",
        globalName: "Pebble",
        discriminator: "0000",
        avatar: "",
        banner: "",
        accentColor: 0,
        bot: false,
        system: false,
        publicFlags: 0,
        roles: [MOCK_GUILD.roles.find((r) => r.roleId === "r_free")!],
    },
];

export function getRoleName(roleId: string) {
    return MOCK_GUILD.roles.find((r) => r.roleId === roleId)?.name ?? roleId;
}
