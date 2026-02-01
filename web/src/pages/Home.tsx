import { RefreshCwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import MemoEditor from "@/components/MemoEditor";
import MemoFilters from "@/components/MemoFilters";
import MemoView from "@/components/MemoView";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useInstance } from "@/contexts/InstanceContext";
import { DEFAULT_LIST_MEMOS_PAGE_SIZE } from "@/helpers/consts";
import { useMemoFilters as useMemoFilterBuilder, useMemoSorting } from "@/hooks";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useInfiniteMemos } from "@/hooks/useMemoQueries";
import { groupMemosSemantically, onSemanticGroupsUpdate, updateSemanticGroupsAsync, type MemoEmbedding } from "@/lib/embeddings";
import { State } from "@/types/proto/api/v1/common_pb";
import { useTranslate } from "@/utils/i18n";

const Home = () => {
  const t = useTranslate();
  const user = useCurrentUser();
  const { isInitialized } = useInstance();
  const [groupMode, setGroupMode] = useState<"normal" | "semantic">("normal");

  const memoFilter = useMemoFilterBuilder({
    creatorName: user?.name,
    includeShortcuts: true,
    includePinned: true,
  });

  const { listSort, orderBy } = useMemoSorting({
    pinnedFirst: true,
    state: State.NORMAL,
  });

  const { data } = useInfiniteMemos(
    {
      state: State.NORMAL,
      orderBy: orderBy || "display_time desc",
      filter: memoFilter,
      pageSize: DEFAULT_LIST_MEMOS_PAGE_SIZE,
    },
    { enabled: isInitialized && !!user },
  );

  const memos = data?.pages.flatMap((page) => page.memos) || [];
  const sortedMemoList = listSort ? listSort(memos) : memos;

  return (
    <div className="w-full h-screen flex flex-col bg-background text-foreground">
      {/* Top panel - MemoEditor */}
      <div className="p-4 flex justify-center">
        <div className="w-1/2">
          <MemoEditor className="mb-2" cacheKey="home-memo-editor" placeholder={t("editor.any-thoughts")} />
        </div>
      </div>

      {/* Tab navigation - center */}
      <div className="flex justify-center items-center py-3 bg-background">
        <ToggleGroup
          type="single"
          value={groupMode}
          onValueChange={(value) => value && setGroupMode(value as "normal" | "semantic")}
          variant="outline"
          className="bg-background shadow-sm border rounded-lg p-1"
        >
          <ToggleGroupItem value="normal" className="px-6 py-2">Normal</ToggleGroupItem>
          <ToggleGroupItem value="semantic" className="px-6 py-2">Semantic</ToggleGroupItem>
        </ToggleGroup>
      </div>

      {/* Bottom panel - shows content based on selected tab */}
      <div className="flex-1 overflow-y-auto p-4 flex justify-center shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)]">
        {groupMode === "normal" ? (
          <div className="w-1/2">
            <div className="flex flex-col gap-2">
              <MemoFilters />
              {sortedMemoList.map((memo) => (
                <MemoView key={`${memo.name}-${memo.displayTime}`} memo={memo} showVisibility showPinned compact />
              ))}
            </div>
          </div>
        ) : (
          <div className="w-full px-4">
            <SemanticPanel />
          </div>
        )}
      </div>
    </div>
  );
};

// Morandi color palette - muted, elegant tones
const MORANDI_COLORS = [
  { bg: "bg-[#a49b8f]", border: "border-[#8a8279]", tag: "bg-[#8a8279]", tagText: "text-white" }, // Warm gray
  { bg: "bg-[#b5c4b1]", border: "border-[#9aad95]", tag: "bg-[#9aad95]", tagText: "text-white" }, // Sage green
  { bg: "bg-[#c9b1a0]", border: "border-[#b39a87]", tag: "bg-[#b39a87]", tagText: "text-white" }, // Dusty rose
  { bg: "bg-[#a7b5c4]", border: "border-[#8fa0b3]", tag: "bg-[#8fa0b3]", tagText: "text-white" }, // Steel blue
  { bg: "bg-[#c4b7a6]", border: "border-[#b0a08c]", tag: "bg-[#b0a08c]", tagText: "text-white" }, // Taupe
  { bg: "bg-[#b8a9c4]", border: "border-[#a192b3]", tag: "bg-[#a192b3]", tagText: "text-white" }, // Dusty lavender
  { bg: "bg-[#c4a9a9]", border: "border-[#b39292]", tag: "bg-[#b39292]", tagText: "text-white" }, // Muted mauve
  { bg: "bg-[#a9c4b8]", border: "border-[#92b3a1]", tag: "bg-[#92b3a1]", tagText: "text-white" }, // Soft teal
];

// Row span patterns for masonry effect
const ROW_SPANS = ["row-span-2", "row-span-3", "row-span-2", "row-span-4", "row-span-3", "row-span-2", "row-span-3", "row-span-4"];

// Generate a readable tag name from memo content
function generateTagName(memos: MemoEmbedding[]): string {
  // Get common words from all memos, excluding common stop words
  const stopWords = new Set(["the", "a", "an", "is", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does", "did", "will", "would", "could", "should", "may", "might", "must", "shall", "can", "need", "dare", "ought", "used", "to", "of", "in", "for", "on", "with", "at", "by", "from", "as", "into", "through", "during", "before", "after", "above", "below", "between", "under", "again", "further", "then", "once", "here", "there", "when", "where", "why", "how", "all", "each", "few", "more", "most", "other", "some", "such", "no", "nor", "not", "only", "own", "same", "so", "than", "too", "very", "just", "and", "but", "if", "or", "because", "until", "while", "although", "though", "after", "before", "when", "whenever", "where", "wherever", "whether", "which", "while", "who", "whom", "whose", "this", "that", "these", "those", "i", "me", "my", "myself", "we", "our", "ours", "ourselves", "you", "your", "yours", "yourself", "yourselves", "he", "him", "his", "himself", "she", "her", "hers", "herself", "it", "its", "itself", "they", "them", "their", "theirs", "themselves", "what"]);

  const wordCounts = new Map<string, number>();

  for (const memo of memos) {
    const words = memo.content.toLowerCase().split(/\s+/);
    for (const word of words) {
      const cleaned = word.replace(/[^a-z0-9]/g, "");
      if (cleaned.length > 2 && !stopWords.has(cleaned)) {
        wordCounts.set(cleaned, (wordCounts.get(cleaned) || 0) + 1);
      }
    }
  }

  // Get top 2 most common words
  const sortedWords = Array.from(wordCounts.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 2)
    .map(([word]) => word.charAt(0).toUpperCase() + word.slice(1));

  return sortedWords.length > 0 ? sortedWords.join(" & ") : "Related Notes";
}

const SemanticPanel = () => {
  const [groups, setGroups] = useState<Map<string, MemoEmbedding[]>>(new Map());
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Initial fetch
  useEffect(() => {
    const fetchGroups = async () => {
      try {
        setLoading(true);
        setError(null);
        console.log("Fetching semantic groups...");
        const semanticGroups = await groupMemosSemantically();
        console.log("Semantic groups fetched:", semanticGroups.size);
        setGroups(semanticGroups);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : String(err);
        setError(`Failed to load semantic groups: ${errorMessage}`);
        console.error("Error loading semantic groups:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchGroups();
  }, []);

  // Listen for automatic updates when notes are saved
  useEffect(() => {
    const unsubscribe = onSemanticGroupsUpdate((updatedGroups) => {
      console.log("Semantic groups updated automatically");
      setGroups(updatedGroups);
      setLoading(false);
      setRefreshing(false);
    });

    return () => unsubscribe();
  }, []);

  // Manual refresh handler
  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await updateSemanticGroupsAsync();
    } catch (err) {
      console.error("Failed to refresh groups:", err);
      setRefreshing(false);
    }
  };

  if (loading) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center p-8">
        <div className="animate-pulse flex flex-col items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-muted" />
          <p className="text-sm text-muted-foreground">Loading semantic groups...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center p-8">
        <div className="text-center text-destructive">
          <p className="text-sm">{error}</p>
        </div>
      </div>
    );
  }

  if (groups.size === 0) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center p-8">
        <div className="text-center text-muted-foreground">
          <p className="text-lg font-medium mb-2">Semantic Grouping</p>
          <p className="text-sm">No memos with embeddings yet. Save some notes to see them grouped semantically.</p>
        </div>
      </div>
    );
  }

  const groupsArray = Array.from(groups.entries());

  return (
    <div className="w-full">
      {/* Refresh button */}
      <div className="flex justify-end mb-4">
        <Button
          variant="outline"
          size="sm"
          onClick={handleRefresh}
          disabled={refreshing}
          className="gap-2"
        >
          <RefreshCwIcon className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
          {refreshing ? "Regrouping..." : "Regroup"}
        </Button>
      </div>

      <div className="grid grid-cols-2 auto-rows-[minmax(80px,auto)] gap-5">
      {groupsArray.map(([_, memos], index) => {
        const colorScheme = MORANDI_COLORS[index % MORANDI_COLORS.length];
        const rowSpan = ROW_SPANS[index % ROW_SPANS.length];
        const tagName = generateTagName(memos);

        return (
          <div
            key={index}
            className={`${rowSpan} rounded-3xl border ${colorScheme.bg} ${colorScheme.border} shadow-md hover:shadow-xl hover:scale-[1.02] transition-all duration-300 ease-out overflow-hidden flex flex-col`}
          >
            {/* Tag header */}
            <div className={`px-5 py-4 ${colorScheme.tag}`}>
              <h3 className={`text-xl font-bold ${colorScheme.tagText} tracking-wide drop-shadow-sm`}>
                {tagName}
              </h3>
              <p className={`text-sm ${colorScheme.tagText} opacity-90 mt-1 font-medium`}>
                {memos.length} {memos.length === 1 ? "note" : "notes"}
              </p>
            </div>

            {/* Notes list */}
            <div className="p-4 flex flex-col gap-3 flex-1 overflow-y-auto">
              {memos.map((memo) => (
                <div
                  key={memo.memo_name}
                  className="bg-white/40 dark:bg-white/10 backdrop-blur-sm rounded-2xl p-4 border border-white/60 dark:border-white/20 shadow-sm hover:bg-white/60 dark:hover:bg-white/20 transition-colors duration-200"
                >
                  <pre className="text-sm text-gray-800 dark:text-gray-100 leading-relaxed whitespace-pre-wrap font-sans">{memo.content}</pre>
                </div>
              ))}
            </div>
          </div>
        );
      })}
      </div>
    </div>
  );
};

export default Home;
