export type DiscordRole = {
    roleId: string;
    name: string;
    color: number;
    managed: boolean;
    mentionable: boolean;
    hoist: boolean;
    position: number;
    permissions: number;
    icon: string;
    unicodeEmoji: string;
    flags: number;
};

export type DiscordChannel = { id: string; name: string };

export type DiscordUser = {
    userId: string;
    username: string;
    globalName: string;
    discriminator: string;
    avatar: string;
    banner: string;
    accentColor: number;
    bot: boolean;
    system: boolean;
    publicFlags: number;
    roles?: DiscordRole[];
    isAuthor?: boolean;
    isAdmin?: boolean;
};

export type Guild = {
    id: string;
    name: string;
    roles: DiscordRole[];
    channels: DiscordChannel[];
};

export type ImageBlob = {
    ID: number;
    imageId: number;
    index: number;
    data: string; // Base64
    contentType: string;
};

export type Image = {
    ID: number;
    postId: number;
    thumbnail: string; // Base64
    blobs: ImageBlob[];
};

export type Post = {
    ID?: number; // Internal
    postKey: string;
    channelId: string;
    guildId: string;
    title: string;
    description: string;
    timestamp: string; // RFC3339
    isPremium: boolean;
    authorId: number;
    author: DiscordUser;
    allowedRoles: DiscordRole[];
    image: Image;
};

export type ViewMode = "gallery" | "post";
export type FileRef = {
    name: string;
    mime: string;
    url: string;
    size: number;
};
