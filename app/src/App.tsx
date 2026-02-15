import { useEffect, useMemo, useState } from "react";
import { intersect, cn } from "./lib/utils";
import { UI } from "./constants";
import { MOCK_GUILD } from "./data/mock";
import type { DiscordUser, Guild, Post, ViewMode } from "./types";
import { DiagonalSlitHeader } from "./components/DiagonalSlitHeader";
import { LoginModal } from "./components/LoginModal";
import { AuthorPanel } from "./components/AuthorPanel";
import { TopBar } from "./components/TopBar";
import { MainGalleryView } from "./components/MainGalleryView";
import { PostDetailView } from "./components/PostDetailView";
import { RightSidebar } from "./components/RightSidebar";
import { ProfileSidebar } from "./components/ProfileSidebar";

function App() {
  const [user, setUser] = useState<DiscordUser | null>(null);
  const [guild] = useState<Guild>(MOCK_GUILD);
  const [view, setView] = useState<ViewMode>("gallery");
  const [loginOpen, setLoginOpen] = useState(false);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);

  // Parse URL and restore state from history
  useEffect(() => {
    const path = window.location.pathname;
    const params = new URLSearchParams(window.location.search);
    const tokenParam = params.get("token");
    const postParam = params.get("post");

    // Handle JWT token from OAuth callback
    if (tokenParam) {
      localStorage.setItem("jwt", tokenParam);
      window.history.replaceState({}, document.title, "/");
    }

    // Handle post ID from URL path or query param
    const pathMatch = path.match(/^\/post\/(.+)$/);
    if (pathMatch) {
      setSelectedId(pathMatch[1]);
      setView("post");
    } else if (postParam) {
      setSelectedId(postParam);
      setView("post");
      window.history.replaceState({}, document.title, `/post/${postParam}`);
    }

    // Parse JWT for user info
    const token = localStorage.getItem("jwt");
    if (token) {
      try {
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function (c) {
          return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));

        const claims = JSON.parse(jsonPayload);
        const u: DiscordUser = {
          userId: claims.uid,
          username: claims.sub,
          globalName: claims.sub,
          discriminator: "0000",
          avatar: claims.avt || "",
          banner: "",
          accentColor: 0,
          bot: false,
          system: false,
          publicFlags: 0,
          roles: claims.roles ? claims.roles.map((r: any) => ({
            id: r.id,
            name: r.name,
            color: r.color,
            managed: r.managed,
            mentionable: r.mentionable,
            hoist: r.hoist,
            position: r.position,
            permissions: r.permissions,
            icon: r.icon,
            unicodeEmoji: r.unicodeEmoji,
            flags: r.flags
          })) : [],
          isAdmin: claims.adm,
          isAuthor: claims.adm
        };
        setUser(u);
      } catch (e) {
        console.error("Failed to parse JWT", e);
        localStorage.removeItem("jwt");
      }
    }
  }, []);

  // Fetch posts
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const limit = 50;

  const loadPosts = (pageNum: number, reset: boolean = false) => {
    setLoading(true);
    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    fetch(`/posts?page=${pageNum}&limit=${limit}`, { headers })
      .then(res => res.json())
      .then(data => {
        setPosts(prev => reset ? data : [...prev, ...data]);
        setHasMore(data.length === limit);
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch posts", err);
        setLoading(false);
      });
  };

  // Initial Fetch
  useEffect(() => {
    loadPosts(1, true);
    setPage(1);
  }, [user]);

  const handleLoadMore = () => {
    if (!hasMore || loading) return;
    const nextPage = page + 1;
    setPage(nextPage);
    loadPosts(nextPage);
  };

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [tagFilter, setTagFilter] = useState<string | null>(null);
  const [q, setQ] = useState("");
  const [transitionRect, setTransitionRect] = useState<DOMRect | null>(null);

  // Handle browser back/forward buttons
  useEffect(() => {
    const handlePopState = () => {
      const path = window.location.pathname;
      const pathMatch = path.match(/^\/post\/(.+)$/);
      
      if (pathMatch) {
        setSelectedId(pathMatch[1]);
        setView("post");
      } else {
        setSelectedId(null);
        setView("gallery");
      }
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  // Reset selectedId when posts load if none selected
  useEffect(() => {
    if (!selectedId && posts.length > 0) {
    }
  }, [posts, selectedId]);

  const selected = useMemo(() => posts?.find((p) => p.postKey === selectedId) ?? null, [posts, selectedId]);

  const viewerRoleIds = useMemo(() => (user?.roles ?? []).map((r) => r.id), [user]);

  const filteredPosts = useMemo(() => {
    const base = posts.slice().sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    const byTag = base; // Tag filtering logic if tags field exists in future
    const s = q.trim().toLowerCase();
    if (!s) return byTag;
    return byTag.filter((p) => {
      const hay = `${p.title ?? ""} ${(p.description ?? "").slice(0, 120)}`.toLowerCase();
      return hay.includes(s);
    });
  }, [posts, tagFilter, q]);

  const canAccessPost = useMemo(() => {
    return (p: Post) => {
      if (user?.isAdmin) return true; // Admin/Author access
      const postRoleIds = p.allowedRoles.map(r => r.id);
      if (postRoleIds.length === 0) return true;
      return intersect(postRoleIds, viewerRoleIds);
    };
  }, [user, viewerRoleIds]);



  async function handleCreate(postInput: {
    title: string;
    description: string;
    allowedRoleIds: string[];
    channelIds: string[];
    image: File;
    thumbnail?: File;
  }) {
    const formData = new FormData();
    formData.append("title", postInput.title);
    formData.append("description", postInput.description);
    formData.append("roles", postInput.allowedRoleIds.join(","));
    formData.append("channels", postInput.channelIds.join(","));
    formData.append("image", postInput.image);

    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

    try {
      const res = await fetch("/posts", {
        method: "POST",
        headers,
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Upload failed: ${err.error || res.statusText}`);
        return;
      }

      const created = await res.json();
      console.log("Post created:", created);
      // Refresh posts to show the new one
      loadPosts(1, true);
      setPage(1);
    } catch (err) {
      console.error("Upload error:", err);
      alert("Upload failed. Check the console for details.");
    }
  }

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
              setSelectedId(pick.postKey);
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

        {user?.isAdmin ? (
          <div className="mt-6">
            <AuthorPanel user={user} onCreate={handleCreate} />
          </div>
        ) : null}

        <div className="mt-6 flex flex-col gap-4 lg:flex-row relative items-start">
          {/* Left: MAIN */}
          <div className={cn("transition-all duration-500 w-full lg:w-[calc(100%-22rem)]")}>
            {loading ? (
              <div className="p-12 text-center font-bold text-zinc-400">Loading posts...</div>
            ) : view === "gallery" ? (
              <MainGalleryView
                posts={filteredPosts}
                selectedId={selectedId}
                canAccessPost={canAccessPost}
                onOpenPost={(id, rect) => {
                  setSelectedId(id);
                  setTransitionRect(rect || null);
                  setView("post");
                  window.history.pushState({ postId: id }, "", `/post/${id}`);
                }}
                q={q}
                setQ={setQ}
                onLoadMore={handleLoadMore}
                hasMore={hasMore}
                loading={loading}
              />
            ) : (
              <PostDetailView
                selected={selected}
                onBack={() => {
                  setView("gallery");
                  setSelectedId(null);
                  window.history.pushState({}, "", "/");
                }}
                transitionRect={transitionRect}
                user={user}
              />
            )}
          </div>

          {/* Right: SIDEBAR */}
          <div
            className={cn(
              "hidden w-80 lg:block"
            )}
          >
            <div className="sticky top-4">
              {view === "gallery" ? (
                <ProfileSidebar user={user} onLogin={() => setLoginOpen(true)} />
              ) : (
                <RightSidebar
                  posts={filteredPosts}
                  selectedId={selectedId}
                  canAccessPost={canAccessPost}
                  onSelect={(id) => {
                    setSelectedId(id);
                    setView("post");
                    window.history.pushState({ postId: id }, "", `/post/${id}`);
                  }}
                  selected={selected}
                  user={user}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
