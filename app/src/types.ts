export type DiscordRole = {
    id: string; // was roleId
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
    size: number;
    filename: string;
};

export type Image = {
    ID: number;
    postId: number;
    thumbnail: string; // Base64
    hasThumbnail?: boolean;
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
    focusX?: number;   // 0-100, default 50
    focusY?: number;   // 0-100, default 50
    authorId: number;
    author: DiscordUser;
    allowedRoles: DiscordRole[];
    images: Image[];
};

export interface Theme {
    border_radius: string;
    border_size: string;

    // Light
    primary_color_light: string;
    secondary_color_light: string;
    page_bg_light: string;
    page_bg_trans_light: number;
    card_bg_light: string;
    card_bg_trans_light: number;
    border_color_light: string;

    // Dark
    primary_color_dark: string;
    secondary_color_dark: string;
    page_bg_dark: string;
    page_bg_trans_dark: number;
    card_bg_dark: string;
    card_bg_trans_dark: number;
    border_color_dark: string;
}

export interface Settings {
    hero_title: string;
    hero_subtitle: string;
    hero_description: string;
    public_access?: boolean;
    theme?: Theme;
}

export type ViewMode = "gallery" | "post" | "not-found";
export type FileRef = {
    name: string;
    mime: string;
    url: string;
    size: number;
};
