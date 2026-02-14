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

  // Parse user from URL query param if present
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tokenParam = params.get("token");

    if (tokenParam) {
      localStorage.setItem("jwt", tokenParam);
      window.history.replaceState({}, document.title, "/");
    }

    const token = localStorage.getItem("jwt");
    if (token) {
      try {
        // Simple parse of JWT payload (2nd part)
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function (c) {
          return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));

        const claims = JSON.parse(jsonPayload);
        // Construct user from claims
        // Claims: uid, sub, adm, roles
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
          roles: claims.roles ? claims.roles.map((r: string) => ({ roleId: r, name: "Role " + r, color: 0 })) : [], // Partial role info
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

    fetch(`http://localhost:3000/posts?page=${pageNum}&limit=${limit}`, { headers })
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

  // Reset selectedId when posts load if none selected
  useEffect(() => {
    if (!selectedId && posts.length > 0) {
    }
  }, [posts, selectedId]);

  const selected = useMemo(() => posts?.find((p) => p.postKey === selectedId) ?? null, [posts, selectedId]);

  const viewerRoleIds = useMemo(() => (user?.roles ?? []).map((r) => r.roleId), [user]);

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
      const postRoleIds = p.allowedRoles.map(r => r.roleId);
      if (postRoleIds.length === 0) return true;
      return intersect(postRoleIds, viewerRoleIds);
    };
  }, [user, viewerRoleIds]);

  const canAccessSelected = useMemo(() => {
    if (!selected) return false;
    return canAccessPost(selected);
  }, [selected, canAccessPost]);

  const accessLabelForSelected = useMemo(() => {
    if (!selected) return "";
    return selected.allowedRoles.length ? `Requires: ${selected.allowedRoles.map(r => r.name).join(", ")}` : "Public";
  }, [selected]);

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
      const res = await fetch("http://localhost:3000/posts", {
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
                canAccessSelected={canAccessSelected}
                accessLabel={accessLabelForSelected}
                onBack={() => setView("gallery")}
                tagFilter={tagFilter}
                setTagFilter={setTagFilter}
                similarPosts={filteredPosts}
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
          <div
            className={cn(
              "fixed inset-y-0 right-0 z-40 w-80 bg-white/95 shadow-2xl backdrop-blur-sm lg:static lg:bg-transparent lg:shadow-none lg:backdrop-blur-none",
              "transition-all duration-500 ease-in-out transform",
              "translate-x-0 opacity-100"
            )}
          >
            <div className="h-full overflow-y-auto p-4 lg:h-auto lg:overflow-visible lg:p-0 lg:sticky lg:top-4">
              {view === "gallery" ? (
                <ProfileSidebar user={user} onLogin={() => setLoginOpen(true)} />
              ) : (
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
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
