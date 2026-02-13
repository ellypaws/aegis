import { useEffect, useMemo, useState } from "react";
import { safeRevoke, intersect, uid } from "./lib/utils";
import { UI } from "./constants";
import { MOCK_GUILD, getRoleName } from "./data/mock";
import type { DiscordUser, Guild, Post, ViewMode, FileRef } from "./types";
import { DiagonalSlitHeader } from "./components/DiagonalSlitHeader";
import { LoginModal } from "./components/LoginModal";
import { AuthorPanel } from "./components/AuthorPanel";
import { TopBar } from "./components/TopBar";
import { MainGalleryView } from "./components/MainGalleryView";
import { PostDetailView } from "./components/PostDetailView";
import { RightSidebar } from "./components/RightSidebar";

function App() {
  const [user, setUser] = useState<DiscordUser | null>(null);
  const [guild] = useState<Guild>(MOCK_GUILD);
  const [view, setView] = useState<ViewMode>("gallery");
  const [loginOpen, setLoginOpen] = useState(false);

  const [posts, setPosts] = useState<Post[]>(() => {
    const mkPlaceholder = (name: string) => {
      const svg = encodeURIComponent(
        `<svg xmlns='http://www.w3.org/2000/svg' width='1024' height='1024'>
          <defs>
            <linearGradient id='g' x1='0' y1='0' x2='1' y2='1'>
              <stop offset='0%' stop-color='${name === "A" ? "#60a5fa" : name === "B" ? "#34d399" : "#f472b6"}'/>
              <stop offset='100%' stop-color='${name === "A" ? "#a78bfa" : name === "B" ? "#fb7185" : "#fbbf24"}'/>
            </linearGradient>
          </defs>
          <rect width='1024' height='1024' fill='url(#g)'/>
          <text x='50%' y='52%' dominant-baseline='middle' text-anchor='middle' fill='rgba(255,255,255,0.85)' font-family='ui-sans-serif' font-size='140' font-weight='700'>${name}</text>
        </svg>`
      );
      const url = `data:image/svg+xml;charset=utf-8,${svg}`;
      return { url, name: `${name}.svg`, mime: "image/svg+xml", size: 1024 * 30 } as FileRef;
    };

    const now = Date.now();
    return [
      {
        id: "p1",
        createdAt: now - 1000 * 60 * 60 * 10,
        title: "VIP Drop A",
        description: "High-res pack for VIP. Thumbnail shown for non-access.",
        tags: ["vip", "pack", "cute"],
        allowedRoleIds: ["r_vip"],
        channelIds: ["c_vip"],
        full: mkPlaceholder("A"),
        thumb: mkPlaceholder("a"),
        authorId: "u_elly",
      },
      {
        id: "p2",
        createdAt: now - 1000 * 60 * 60 * 6,
        title: "Tier 1 Sketches",
        description: "Tier 1 access. No thumbnail, so locked users see the lock text.",
        tags: ["sketch", "tier1"],
        allowedRoleIds: ["r_tier1", "r_tier2", "r_vip"],
        channelIds: ["c_art"],
        full: mkPlaceholder("B"),
        thumb: undefined,
        authorId: "u_elly",
      },
      {
        id: "p3",
        createdAt: now - 1000 * 60 * 60 * 2,
        title: "Tier 2 Color",
        description: "Tier 2+ full image; everyone sees the blurred thumbnail.",
        tags: ["color", "tier2"],
        allowedRoleIds: ["r_tier2", "r_vip"],
        channelIds: ["c_art", "c_ann"],
        full: mkPlaceholder("C"),
        thumb: mkPlaceholder("c"),
        authorId: "u_elly",
      },
    ];
  });

  const [selectedId, setSelectedId] = useState<string | null>(posts[0]?.id ?? null);
  const [tagFilter, setTagFilter] = useState<string | null>(null);
  const [q, setQ] = useState("");
  const [transitionRect, setTransitionRect] = useState<DOMRect | null>(null);

  const selected = useMemo(() => posts.find((p) => p.id === selectedId) ?? posts[0] ?? null, [posts, selectedId]);

  useEffect(() => {
    if (!selectedId && posts[0]) setSelectedId(posts[0].id);
  }, [posts, selectedId]);

  const viewerRoleIds = useMemo(() => (user?.roles ?? []).map((r) => r.id), [user]);

  const filteredPosts = useMemo(() => {
    const base = posts.slice().sort((a, b) => b.createdAt - a.createdAt);
    const byTag = tagFilter ? base.filter((p) => p.tags.includes(tagFilter)) : base;
    const s = q.trim().toLowerCase();
    if (!s) return byTag;
    return byTag.filter((p) => {
      const hay = `${p.title ?? ""} ${(p.description ?? "").slice(0, 120)} ${p.tags.join(" ")}`.toLowerCase();
      return hay.includes(s);
    });
  }, [posts, tagFilter, q]);

  const canAccessPost = useMemo(() => {
    return (p: Post) => {
      if (user?.isAuthor && user.id === p.authorId) return true;
      return intersect(p.allowedRoleIds, viewerRoleIds);
    };
  }, [user, viewerRoleIds]);

  const canAccessSelected = useMemo(() => {
    if (!selected) return false;
    return canAccessPost(selected);
  }, [selected, canAccessPost]);

  const accessLabelForSelected = useMemo(() => {
    if (!selected) return "";
    return selected.allowedRoleIds.length ? `Requires: ${selected.allowedRoleIds.map(getRoleName).join(", ")}` : "No roles configured";
  }, [selected]);

  function handleCreate(postInput: Omit<Post, "id" | "createdAt">) {
    const post: Post = { ...postInput, id: uid(), createdAt: Date.now() };
    setPosts((p) => [post, ...p]);
    setSelectedId(post.id);
    setTagFilter(null);
    setQ("");
    setView("post");
    setTransitionRect(null);
  }

  // Cleanup blob URLs on unmount.
  useEffect(() => {
    return () => {
      for (const p of posts) {
        if (p.full.url.startsWith("blob:")) safeRevoke(p.full.url);
        if (p.thumb?.url?.startsWith("blob:")) safeRevoke(p.thumb.url);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className={UI.page}>
      <LoginModal
        open={loginOpen}
        onClose={() => setLoginOpen(false)}
        onLogin={(u) => {
          setUser(u);
        }}
      />

      <div className={UI.max}>
        <DiagonalSlitHeader
          posts={posts}
          onClickRandom={() => {
            const pick = filteredPosts[Math.floor(Math.random() * Math.max(1, filteredPosts.length))];
            if (pick) {
              setSelectedId(pick.id);
              setView("post");
            }
          }}
        />

        <TopBar
          guild={guild}
          view={view}
          setView={setView}
          tagFilter={tagFilter}
          setTagFilter={setTagFilter}
          user={user}
          setLoginOpen={setLoginOpen}
          setUser={setUser}
          selectedId={selectedId}
        />

        {user?.isAuthor ? (
          <div className="mt-6">
            <AuthorPanel user={user} onCreate={handleCreate} />
          </div>
        ) : null}

        <div className="mt-6 grid gap-4 lg:grid-cols-12">
          {/* Left: MAIN */}
          <div className="lg:col-span-8">
            {view === "gallery" ? (
              <MainGalleryView
                posts={filteredPosts}
                selectedId={selectedId}
                canAccessPost={canAccessPost}
                onOpenPost={(id, rect) => {
                  setSelectedId(id);
                  setTransitionRect(rect || null);
                  setView("post");
                }}
                q={q}
                setQ={setQ}
              />
            ) : (
              <PostDetailView
                selected={selected}
                canAccessSelected={canAccessSelected}
                accessLabel={accessLabelForSelected}
                onBack={() => setView("gallery")}
                tagFilter={tagFilter}
                setTagFilter={setTagFilter}
                similarPosts={filteredPosts.filter((p) => tagFilter && p.tags.includes(tagFilter))}
                canAccessPost={canAccessPost}
                onSelectSimilar={(id) => {
                  setSelectedId(id);
                  setTransitionRect(null);
                  setView("post");
                }}
                transitionRect={transitionRect}
              />
            )}
          </div>

          {/* Right: SIDEBAR */}
          <div className="lg:col-span-4">
            <RightSidebar
              tagFilter={tagFilter}
              setTagFilter={setTagFilter}
              q={q}
              posts={filteredPosts}
              selectedId={selectedId}
              canAccessPost={canAccessPost}
              onSelect={(id) => {
                setSelectedId(id);
                setView("post");
              }}
              selected={selected}
              canAccessSelected={canAccessSelected}
              user={user}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
