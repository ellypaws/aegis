export type DiscordRole = { id: string; name: string; color?: string };
export type DiscordChannel = { id: string; name: string };
export type DiscordUser = {
    id: string;
    username: string;
    avatarUrl?: string;
    roles: DiscordRole[];
    isAuthor?: boolean;
};

export type FileRef = {
    name: string;
    mime: string;
    url: string;
    size: number;
};

export type Post = {
    id: string;
    createdAt: number;
    title?: string;
    description?: string;
    tags: string[];
    allowedRoleIds: string[];
    channelIds: string[];
    full: FileRef;
    thumb?: FileRef;
    authorId: string;
};

export type Guild = {
    id: string;
    name: string;
    roles: DiscordRole[];
    channels: DiscordChannel[];
};

export type ViewMode = "gallery" | "post";
