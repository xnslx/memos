import OpenAI from 'openai';
import { supabase, type MemoEmbedding } from './supabase';

export type { MemoEmbedding };

// =============================================================================
// Configuration
// =============================================================================

const CONFIG = {
  // Embedding provider: "openai" or "huggingface"
  EMBEDDING_PROVIDER: 'huggingface' as 'openai' | 'huggingface',

  // OpenAI config (1536 dimensions)
  OPENAI_API_KEY: import.meta.env.VITE_OPENAI_API_KEY || '',

  // Hugging Face config (384 dimensions)
  HF_MODEL: 'sentence-transformers/all-MiniLM-L6-v2',
  HF_TOKEN: import.meta.env.VITE_HF_TOKEN || '',

  // kNN parameters
  K_NEIGHBORS: 5, // Number of nearest neighbors to consider
  MUTUAL_KNN_REQUIRED: true, // Only keep edges where both nodes are in each other's kNN

  // Edge pruning
  MIN_SIMILARITY_THRESHOLD: 0.4, // Minimum similarity to consider an edge
  BRIDGE_NODE_MAX_DEGREE: 8, // Nodes with more connections are considered "bridge" nodes
  BRIDGE_NODE_PENALTY: 0.15, // Reduce similarity for bridge node connections

  // Chunking for long notes
  CHUNK_SIZE: 500, // Characters per chunk
  CHUNK_OVERLAP: 50, // Overlap between chunks
  LONG_NOTE_THRESHOLD: 800, // Notes longer than this get chunked/summarized

  // Intent summary (only for OpenAI provider)
  USE_INTENT_SUMMARY: false, // Generate LLM intent summaries for long notes

  // Cluster splitting by variance
  MAX_SIMILARITY_VARIANCE: 0.04, // Split clusters with variance above this (std dev ~0.2)
  MIN_CLUSTER_SIZE_FOR_SPLIT: 3, // Only consider splitting clusters with at least this many memos
  MIN_AVG_SIMILARITY_FOR_COHESION: 0.5, // Minimum average similarity to keep cluster together
};

// Export config for UI components
export function getEmbeddingConfig() {
  return {
    provider: CONFIG.EMBEDDING_PROVIDER,
    dimensions: CONFIG.EMBEDDING_PROVIDER === 'openai' ? 1536 : 384,
  };
}

// OpenAI client
const openai = new OpenAI({
  apiKey: CONFIG.OPENAI_API_KEY,
  dangerouslyAllowBrowser: true,
});

// =============================================================================
// Event Emitter for UI Updates
// =============================================================================

type SemanticUpdateListener = (groups: Map<string, MemoEmbedding[]>) => void;
const listeners: Set<SemanticUpdateListener> = new Set();

export function onSemanticGroupsUpdate(
  listener: SemanticUpdateListener
): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function notifyListeners(groups: Map<string, MemoEmbedding[]>) {
  for (const listener of listeners) {
    listener(groups);
  }
}

// =============================================================================
// Embedding Generation (supports OpenAI and HuggingFace)
// =============================================================================

// Generate embedding for text using configured provider
export async function generateEmbedding(text: string): Promise<number[]> {
  if (CONFIG.EMBEDDING_PROVIDER === 'openai') {
    return generateOpenAIEmbedding(text);
  } else {
    return generateHuggingFaceEmbedding(text);
  }
}

// Generate embedding using OpenAI API
async function generateOpenAIEmbedding(text: string): Promise<number[]> {
  try {
    const response = await openai.embeddings.create({
      model: 'text-embedding-3-small',
      input: text,
    });

    return response.data[0].embedding;
  } catch (error) {
    console.error('OpenAI embedding error:', error);
    throw error;
  }
}

// Generate embedding using HuggingFace via backend proxy (avoids CORS)
async function generateHuggingFaceEmbedding(text: string): Promise<number[]> {
  // Use backend proxy to avoid CORS issues
  const API_URL = '/api/embedding';

  try {
    const response = await fetch(API_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        inputs: text,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Embedding API error: ${response.status} - ${errorText}`);
    }

    const result = await response.json();

    // Handle response format - sentence-transformers returns [384] directly
    if (Array.isArray(result)) {
      // If nested array (token embeddings), mean pool them
      if (Array.isArray(result[0])) {
        const tokens = result as number[][];
        const dims = tokens[0].length;
        const pooled = new Array(dims).fill(0);
        for (const token of tokens) {
          for (let i = 0; i < dims; i++) {
            pooled[i] += token[i];
          }
        }
        return pooled.map((v) => v / tokens.length);
      }
      // Direct embedding array
      return result as number[];
    }

    throw new Error('Unexpected response format from embedding API');
  } catch (error) {
    console.error('Embedding API error:', error);
    throw error;
  }
}

// Generate embeddings for multiple texts (batched)
async function generateEmbeddings(texts: string[]): Promise<number[][]> {
  if (texts.length === 0) return [];

  const results: number[][] = [];
  for (const text of texts) {
    const embedding = await generateEmbedding(text);
    results.push(embedding);
  }
  return results;
}

// =============================================================================
// Text Chunking for Long Notes
// =============================================================================

function chunkText(text: string, chunkSize: number, overlap: number): string[] {
  if (text.length <= chunkSize) {
    return [text];
  }

  const chunks: string[] = [];
  let start = 0;

  while (start < text.length) {
    const end = Math.min(start + chunkSize, text.length);
    chunks.push(text.slice(start, end));
    start += chunkSize - overlap;
  }

  return chunks;
}

// Average multiple embeddings into one
function averageEmbeddings(embeddings: number[][]): number[] {
  if (embeddings.length === 0) return [];
  if (embeddings.length === 1) return embeddings[0];

  const dim = embeddings[0].length;
  const avg = new Array(dim).fill(0);

  for (const emb of embeddings) {
    for (let i = 0; i < dim; i++) {
      avg[i] += emb[i];
    }
  }

  for (let i = 0; i < dim; i++) {
    avg[i] /= embeddings.length;
  }

  // Normalize the averaged embedding
  const norm = Math.sqrt(avg.reduce((sum, v) => sum + v * v, 0));
  return avg.map((v) => v / norm);
}

// =============================================================================
// Intent Summary Generation (placeholder - not used with HuggingFace)
// =============================================================================

// Intent summary generation - returns truncated content (LLM not available)
export function generateIntentSummary(content: string): string {
  return content.slice(0, 200);
}

// =============================================================================
// Save Memo Embedding (with chunking and intent summary)
// =============================================================================

export async function saveMemoEmbedding(
  memoName: string,
  content: string
): Promise<void> {
  console.log('=== saveMemoEmbedding called ===');
  console.log('Memo name:', memoName);
  console.log('Content length:', content.length);

  try {
    let embedding: number[];

    const isLongNote = content.length > CONFIG.LONG_NOTE_THRESHOLD;

    if (isLongNote) {
      // For long notes: chunk and average embeddings
      const chunks = chunkText(
        content,
        CONFIG.CHUNK_SIZE,
        CONFIG.CHUNK_OVERLAP
      );
      console.log(`Long note detected, chunking into ${chunks.length} parts`);

      const chunkEmbeddings = await generateEmbeddings(chunks);
      embedding = averageEmbeddings(chunkEmbeddings);
    } else {
      // For short notes: direct embedding
      console.log(`Generating embedding via ${CONFIG.EMBEDDING_PROVIDER}...`);
      embedding = await generateEmbedding(content);
      console.log('Embedding generated, dimensions:', embedding.length);
    }

    console.log(
      `Saving to Supabase: memo_name=${memoName}, embedding_dims=${embedding.length}`
    );

    // Try to save with basic columns (works with any embedding dimension)
    const { data, error } = await supabase
      .from('memo_embeddings')
      .upsert(
        {
          memo_name: memoName,
          content: content,
          embedding: embedding,
          updated_at: new Date().toISOString(),
        },
        {
          onConflict: 'memo_name',
        }
      )
      .select();

    if (error) {
      console.error('Supabase error:', error);
      throw new Error(`Supabase error: ${error.message}`);
    }

    console.log('Supabase response:', data);
    console.log('Embedding saved successfully!');

    // Automatically trigger semantic grouping after saving
    console.log('Updating semantic groups...');
    await updateSemanticGroupsAsync();
    console.log('Semantic groups updated!');
  } catch (error) {
    console.error('Error in saveMemoEmbedding:', error);
    throw error;
  }
}

// =============================================================================
// Delete Memo Embedding
// =============================================================================

export async function deleteMemoEmbedding(memoName: string): Promise<void> {
  const { error } = await supabase
    .from('memo_embeddings')
    .delete()
    .eq('memo_name', memoName);

  if (error) {
    console.error('Error deleting memo embedding:', error);
    throw error;
  }
}

// =============================================================================
// Get All Memo Embeddings
// =============================================================================

// Parse pgvector string to number array
// pgvector returns embeddings as strings like "[0.1,0.2,0.3,...]"
function parseEmbedding(embedding: unknown): number[] {
  if (!embedding) return [];

  // Already a number array
  if (Array.isArray(embedding) && typeof embedding[0] === 'number') {
    return embedding;
  }

  // String from pgvector - parse it
  if (typeof embedding === 'string') {
    try {
      const parsed = JSON.parse(embedding);
      if (Array.isArray(parsed)) {
        return parsed;
      }
    } catch {
      console.error('Failed to parse embedding string:', embedding.slice(0, 50));
    }
  }

  return [];
}

export async function getAllMemoEmbeddings(): Promise<MemoEmbedding[]> {
  try {
    // Try to fetch all columns including new ones
    const { data, error } = await supabase
      .from('memo_embeddings')
      .select(
        'id, memo_name, content, embedding, intent_summary, intent_embedding, chunk_count, created_at, updated_at'
      )
      .order('created_at', { ascending: false });

    if (error) {
      // If new columns don't exist, fallback to basic columns
      console.warn(
        'Error fetching with new columns, trying basic columns:',
        error.message
      );
      const { data: basicData, error: basicError } = await supabase
        .from('memo_embeddings')
        .select('id, memo_name, content, embedding, created_at, updated_at')
        .order('created_at', { ascending: false });

      if (basicError) {
        console.error('Error fetching memo embeddings:', basicError);
        throw basicError;
      }

      // Parse embeddings from pgvector string format
      return (basicData || []).map((item) => ({
        ...item,
        embedding: parseEmbedding(item.embedding),
      }));
    }

    // Parse embeddings from pgvector string format
    return (data || []).map((item) => ({
      ...item,
      embedding: parseEmbedding(item.embedding),
      intent_embedding: item.intent_embedding ? parseEmbedding(item.intent_embedding) : null,
    }));
  } catch (err) {
    console.error('Error in getAllMemoEmbeddings:', err);
    throw err;
  }
}

// =============================================================================
// kNN Graph via Supabase RPC
// =============================================================================

interface KnnEdge {
  source_memo: string;
  target_memo: string;
  similarity: number;
}

async function fetchKnnGraph(): Promise<KnnEdge[]> {
  try {
    const { data, error } = await supabase.rpc('get_all_knn', {
      k: CONFIG.K_NEIGHBORS,
    });

    if (error) {
      console.warn(
        'RPC get_all_knn not available, using client-side fallback:',
        error.message
      );
      return computeKnnGraphClientSide();
    }

    return data || [];
  } catch (err) {
    console.warn('Failed to call RPC, using client-side fallback:', err);
    return computeKnnGraphClientSide();
  }
}

// Fallback: Compute kNN client-side if RPC is not available
async function computeKnnGraphClientSide(): Promise<KnnEdge[]> {
  const embeddings = await getAllMemoEmbeddings();
  const edges: KnnEdge[] = [];

  for (const source of embeddings) {
    if (!source.embedding) continue; // Skip if no embedding

    const similarities: { target: MemoEmbedding; sim: number }[] = [];

    for (const target of embeddings) {
      if (source.memo_name === target.memo_name) continue;
      if (!target.embedding) continue; // Skip if no embedding

      // Compute similarity using both main embedding and intent embedding
      let sim = cosineSimilarity(source.embedding, target.embedding);

      // If intent embeddings exist, use the max of different combinations
      if (source.intent_embedding && target.intent_embedding) {
        const intentSim = cosineSimilarity(
          source.intent_embedding,
          target.intent_embedding
        );
        const crossSim1 = cosineSimilarity(
          source.embedding,
          target.intent_embedding
        );
        const crossSim2 = cosineSimilarity(
          source.intent_embedding,
          target.embedding
        );
        sim = Math.max(sim, intentSim, crossSim1, crossSim2);
      } else if (source.intent_embedding && target.embedding) {
        sim = Math.max(
          sim,
          cosineSimilarity(source.intent_embedding, target.embedding)
        );
      } else if (source.embedding && target.intent_embedding) {
        sim = Math.max(
          sim,
          cosineSimilarity(source.embedding, target.intent_embedding)
        );
      }

      similarities.push({ target, sim });
    }

    // Sort and take top k
    similarities.sort((a, b) => b.sim - a.sim);
    const topK = similarities.slice(0, CONFIG.K_NEIGHBORS);

    for (const { target, sim } of topK) {
      edges.push({
        source_memo: source.memo_name,
        target_memo: target.memo_name,
        similarity: sim,
      });
    }
  }

  return edges;
}

// Cosine similarity between two vectors
function cosineSimilarity(a: number[], b: number[]): number {
  if (!a || !b || a.length !== b.length) return 0;

  let dotProduct = 0;
  let normA = 0;
  let normB = 0;

  for (let i = 0; i < a.length; i++) {
    dotProduct += a[i] * b[i];
    normA += a[i] * a[i];
    normB += b[i] * b[i];
  }

  const denominator = Math.sqrt(normA) * Math.sqrt(normB);
  return denominator === 0 ? 0 : dotProduct / denominator;
}

// =============================================================================
// Mutual kNN + Edge Pruning
// =============================================================================

interface PrunedEdge {
  source: string;
  target: string;
  weight: number;
}

function buildMutualKnnGraph(edges: KnnEdge[]): PrunedEdge[] {
  // Build adjacency lists
  const knnOf = new Map<string, Set<string>>();
  const similarityMap = new Map<string, number>();

  for (const edge of edges) {
    if (!knnOf.has(edge.source_memo)) {
      knnOf.set(edge.source_memo, new Set());
    }
    knnOf.get(edge.source_memo)!.add(edge.target_memo);

    // Store similarity for the edge
    const key = `${edge.source_memo}|${edge.target_memo}`;
    similarityMap.set(key, edge.similarity);
  }

  // Calculate node degrees for bridge detection
  const nodeDegree = new Map<string, number>();
  for (const edge of edges) {
    nodeDegree.set(
      edge.source_memo,
      (nodeDegree.get(edge.source_memo) || 0) + 1
    );
  }

  // Build mutual kNN edges with pruning
  const mutualEdges: PrunedEdge[] = [];
  const processedPairs = new Set<string>();

  for (const edge of edges) {
    const { source_memo: source, target_memo: target, similarity } = edge;

    // Skip if already processed (undirected)
    const pairKey = [source, target].sort().join('|');
    if (processedPairs.has(pairKey)) continue;
    processedPairs.add(pairKey);

    // Skip edges below minimum threshold
    if (similarity < CONFIG.MIN_SIMILARITY_THRESHOLD) continue;

    // Check for mutual kNN
    const isMutual = knnOf.get(target)?.has(source) || false;

    if (CONFIG.MUTUAL_KNN_REQUIRED && !isMutual) {
      continue; // Skip non-mutual edges
    }

    // Apply bridge node penalty
    let weight = similarity;
    const sourceDegree = nodeDegree.get(source) || 0;
    const targetDegree = nodeDegree.get(target) || 0;

    if (
      sourceDegree > CONFIG.BRIDGE_NODE_MAX_DEGREE ||
      targetDegree > CONFIG.BRIDGE_NODE_MAX_DEGREE
    ) {
      weight -= CONFIG.BRIDGE_NODE_PENALTY;
    }

    // Only keep edges with positive weight after penalties
    if (weight > CONFIG.MIN_SIMILARITY_THRESHOLD) {
      mutualEdges.push({ source, target, weight });
    }
  }

  return mutualEdges;
}

// =============================================================================
// Union-Find for Clustering
// =============================================================================

class UnionFind {
  private parent: Map<string, string>;
  private rank: Map<string, number>;

  constructor(items: string[]) {
    this.parent = new Map();
    this.rank = new Map();
    for (const item of items) {
      this.parent.set(item, item);
      this.rank.set(item, 0);
    }
  }

  find(x: string): string {
    if (this.parent.get(x) !== x) {
      this.parent.set(x, this.find(this.parent.get(x)!));
    }
    return this.parent.get(x)!;
  }

  union(x: string, y: string): void {
    const rootX = this.find(x);
    const rootY = this.find(y);

    if (rootX !== rootY) {
      const rankX = this.rank.get(rootX)!;
      const rankY = this.rank.get(rootY)!;

      if (rankX < rankY) {
        this.parent.set(rootX, rootY);
      } else if (rankX > rankY) {
        this.parent.set(rootY, rootX);
      } else {
        this.parent.set(rootY, rootX);
        this.rank.set(rootX, rankX + 1);
      }
    }
  }

  getGroups(): Map<string, string[]> {
    const groups = new Map<string, string[]>();
    for (const [item] of this.parent) {
      const root = this.find(item);
      if (!groups.has(root)) {
        groups.set(root, []);
      }
      groups.get(root)!.push(item);
    }
    return groups;
  }
}

// =============================================================================
// Cluster Variance Analysis & Splitting
// =============================================================================

interface ClusterStats {
  avgSimilarity: number;
  variance: number;
  minSimilarity: number;
  maxSimilarity: number;
  pairwiseSimilarities: { i: number; j: number; sim: number }[];
}

// Calculate internal similarity statistics for a cluster
function calculateClusterStats(memos: MemoEmbedding[]): ClusterStats {
  if (memos.length < 2) {
    return {
      avgSimilarity: 1,
      variance: 0,
      minSimilarity: 1,
      maxSimilarity: 1,
      pairwiseSimilarities: [],
    };
  }

  const similarities: { i: number; j: number; sim: number }[] = [];

  for (let i = 0; i < memos.length; i++) {
    for (let j = i + 1; j < memos.length; j++) {
      if (!memos[i].embedding || !memos[j].embedding) continue;

      let sim = cosineSimilarity(memos[i].embedding, memos[j].embedding);

      // Also consider intent embeddings if available
      if (memos[i].intent_embedding && memos[j].intent_embedding) {
        const intentSim = cosineSimilarity(
          memos[i].intent_embedding as number[],
          memos[j].intent_embedding as number[]
        );
        sim = Math.max(sim, intentSim);
      }

      similarities.push({ i, j, sim });
    }
  }

  if (similarities.length === 0) {
    return {
      avgSimilarity: 0,
      variance: 0,
      minSimilarity: 0,
      maxSimilarity: 0,
      pairwiseSimilarities: [],
    };
  }

  const sims = similarities.map((s) => s.sim);
  const avgSimilarity = sims.reduce((a, b) => a + b, 0) / sims.length;
  const variance =
    sims.reduce((sum, s) => sum + Math.pow(s - avgSimilarity, 2), 0) /
    sims.length;
  const minSimilarity = Math.min(...sims);
  const maxSimilarity = Math.max(...sims);

  return {
    avgSimilarity,
    variance,
    minSimilarity,
    maxSimilarity,
    pairwiseSimilarities: similarities,
  };
}

// Split a cluster into sub-clusters using spectral-like bisection
function splitClusterByVariance(
  memos: MemoEmbedding[],
  stats: ClusterStats
): MemoEmbedding[][] {
  if (memos.length < 2) return [memos];

  // Find the memo that is most "central" (highest avg similarity to others)
  const avgSimToOthers = memos.map((_, idx) => {
    const simsInvolving = stats.pairwiseSimilarities.filter(
      (s) => s.i === idx || s.j === idx
    );
    return (
      simsInvolving.reduce((sum, s) => sum + s.sim, 0) /
      Math.max(simsInvolving.length, 1)
    );
  });

  // Find the memo that is most "peripheral" (lowest avg similarity to others)
  const centralIdx = avgSimToOthers.indexOf(Math.max(...avgSimToOthers));
  const peripheralIdx = avgSimToOthers.indexOf(Math.min(...avgSimToOthers));

  if (centralIdx === peripheralIdx) return [memos];

  // Use the two most dissimilar memos as seeds for bisection
  const seed1 = memos[centralIdx];
  const seed2 = memos[peripheralIdx];

  const cluster1: MemoEmbedding[] = [];
  const cluster2: MemoEmbedding[] = [];

  for (const memo of memos) {
    const simToSeed1 = cosineSimilarity(memo.embedding, seed1.embedding);
    const simToSeed2 = cosineSimilarity(memo.embedding, seed2.embedding);

    if (simToSeed1 >= simToSeed2) {
      cluster1.push(memo);
    } else {
      cluster2.push(memo);
    }
  }

  // Ensure we actually split (avoid empty clusters)
  if (cluster1.length === 0) return [cluster2];
  if (cluster2.length === 0) return [cluster1];

  return [cluster1, cluster2];
}

// Recursively split clusters that have high variance
function splitHighVarianceClusters(
  clusters: MemoEmbedding[][]
): MemoEmbedding[][] {
  const result: MemoEmbedding[][] = [];

  for (const cluster of clusters) {
    // Skip small clusters
    if (cluster.length < CONFIG.MIN_CLUSTER_SIZE_FOR_SPLIT) {
      result.push(cluster);
      continue;
    }

    const stats = calculateClusterStats(cluster);
    console.log(
      `Cluster (${cluster.length} memos): avg=${stats.avgSimilarity.toFixed(
        3
      )}, var=${stats.variance.toFixed(
        4
      )}, range=[${stats.minSimilarity.toFixed(
        3
      )}, ${stats.maxSimilarity.toFixed(3)}]`
    );

    // Check if cluster needs splitting
    const needsSplit =
      stats.variance > CONFIG.MAX_SIMILARITY_VARIANCE ||
      stats.avgSimilarity < CONFIG.MIN_AVG_SIMILARITY_FOR_COHESION;

    if (needsSplit && cluster.length >= CONFIG.MIN_CLUSTER_SIZE_FOR_SPLIT) {
      console.log(`Splitting cluster due to high variance or low cohesion...`);
      const subClusters = splitClusterByVariance(cluster, stats);

      // Recursively check sub-clusters
      const refinedSubClusters = splitHighVarianceClusters(subClusters);
      result.push(...refinedSubClusters);
    } else {
      result.push(cluster);
    }
  }

  return result;
}

// =============================================================================
// Main Semantic Grouping Algorithm
// =============================================================================

export async function groupMemosSemantically(): Promise<
  Map<string, MemoEmbedding[]>
> {
  console.log('Starting semantic grouping...');

  let embeddings: MemoEmbedding[];
  try {
    embeddings = await getAllMemoEmbeddings();
    console.log(`Fetched ${embeddings.length} embeddings`);
  } catch (err) {
    console.error('Failed to fetch embeddings:', err);
    throw err;
  }

  if (embeddings.length === 0) {
    console.log('No embeddings found, returning empty map');
    return new Map();
  }

  // Use simple threshold-based grouping: similarity >= 0.4 means semantically related
  return simpleSemanticGrouping(embeddings);
}

// Threshold-based grouping: compare each note against all others, similarity >= 0.4 = related
function simpleSemanticGrouping(
  embeddings: MemoEmbedding[]
): Map<string, MemoEmbedding[]> {
  console.log('Using semantic grouping (threshold >= 0.4)...');

  const threshold = 0.4; // Semantic similarity threshold for all-MiniLM-L6-v2
  const result = new Map<string, MemoEmbedding[]>();

  // Filter embeddings that have valid embedding vectors
  const validEmbeddings = embeddings.filter(
    (e) => e.embedding && e.embedding.length > 0
  );

  if (validEmbeddings.length === 0) {
    console.log('No valid embeddings found');
    return new Map();
  }

  // Use Union-Find for transitive grouping
  const uf = new UnionFind(validEmbeddings.map((e) => e.memo_name));

  // Compare all pairs: source vs compared sentences
  let printed = 0;
  const PRINT_LIMIT = 30;

  for (let i = 0; i < validEmbeddings.length; i++) {
    const source = validEmbeddings[i];

    for (let j = i + 1; j < validEmbeddings.length; j++) {
      const compared = validEmbeddings[j];

      // quick visibility of dimension mismatch
      if (printed < PRINT_LIMIT) {
        console.log(
          'COMPARE',
          source.memo_name,
          source.embedding?.length,
          '<->',
          compared.memo_name,
          compared.embedding?.length
        );
        printed++;
      }

      // skip mismatched dims (prevents sim=0 confusion)
      if (source.embedding.length !== compared.embedding.length) continue;

      const sim = cosineSimilarity(source.embedding, compared.embedding);

      if (printed < PRINT_LIMIT) {
        console.log('SIM', sim.toFixed(3));
        printed++;
      }

      if (sim >= threshold) {
        console.log(
          `✅ Related: ${source.memo_name} <-> ${
            compared.memo_name
          } (${sim.toFixed(3)})`
        );
        uf.union(source.memo_name, compared.memo_name);
      }
    }
  }

  // Build groups from Union-Find
  const groups = uf.getGroups();
  const memoMap = new Map(validEmbeddings.map((e) => [e.memo_name, e]));

  let groupIndex = 0;
  for (const [, memoNames] of groups) {
    const groupMemos = memoNames
      .map((name) => memoMap.get(name)!)
      .filter(Boolean);
    if (groupMemos.length > 0) {
      result.set(`Group ${++groupIndex}`, groupMemos);
    }
  }

  console.log(
    `Semantic grouping complete: ${result.size} groups from ${validEmbeddings.length} memos`
  );
  return result;
}

// Advanced grouping with kNN, mutual-kNN, edge pruning, and variance splitting
async function advancedSemanticGrouping(
  embeddings: MemoEmbedding[]
): Promise<Map<string, MemoEmbedding[]>> {
  console.log(
    'Using advanced semantic grouping with kNN + mutual-kNN + edge pruning...'
  );

  // Step 1: Fetch kNN graph (via Supabase RPC or client-side fallback)
  console.log('Fetching kNN graph...');
  const knnEdges = await fetchKnnGraph();
  console.log(`Got ${knnEdges.length} kNN edges`);

  if (knnEdges.length === 0) {
    console.log('No kNN edges, falling back to simple grouping');
    return simpleSemanticGrouping(embeddings);
  }

  // Step 2: Build mutual-kNN graph with edge pruning
  console.log('Building mutual-kNN graph with edge pruning...');
  const prunedEdges = buildMutualKnnGraph(knnEdges);
  console.log(`After pruning: ${prunedEdges.length} edges`);

  // Step 3: Cluster using Union-Find
  const allMemoNames = embeddings.map((e) => e.memo_name);
  const uf = new UnionFind(allMemoNames);

  // Sort edges by weight descending for better clustering
  prunedEdges.sort((a, b) => b.weight - a.weight);

  for (const edge of prunedEdges) {
    uf.union(edge.source, edge.target);
  }

  // Step 4: Build initial clusters
  const memoGroups = uf.getGroups();
  const memoMap = new Map(embeddings.map((e) => [e.memo_name, e]));

  const initialClusters: MemoEmbedding[][] = [];
  for (const [_, memoNames] of memoGroups) {
    const groupMemos = memoNames
      .map((name) => memoMap.get(name)!)
      .filter(Boolean);
    if (groupMemos.length > 0) {
      initialClusters.push(groupMemos);
    }
  }

  console.log(`Initial clustering: ${initialClusters.length} clusters`);

  // Step 5: Split clusters with high internal variance
  console.log('Analyzing cluster variance and splitting if needed...');
  const refinedClusters = splitHighVarianceClusters(initialClusters);
  console.log(
    `After variance-based splitting: ${refinedClusters.length} clusters`
  );

  // Step 6: Build result map
  const result = new Map<string, MemoEmbedding[]>();
  let groupIndex = 0;

  for (const cluster of refinedClusters) {
    if (cluster.length > 0) {
      const groupName = `Group ${++groupIndex}`;
      result.set(groupName, cluster);
    }
  }

  console.log(`Advanced grouping complete: ${result.size} groups`);
  return result;
}

// =============================================================================
// Update Semantic Groups (with notification)
// =============================================================================

export async function updateSemanticGroupsAsync(): Promise<
  Map<string, MemoEmbedding[]>
> {
  console.log('Updating semantic groups...');

  try {
    const groups = await groupMemosSemantically();
    console.log(`Found ${groups.size} semantic groups`);

    // Notify all listeners about the updated groups
    notifyListeners(groups);

    return groups;
  } catch (error) {
    console.error('Error updating semantic groups:', error);
    throw error;
  }
}

// =============================================================================
// Legacy: Find Similar Memos
// =============================================================================

export async function findSimilarMemos(
  memoName: string,
  limit = 5
): Promise<MemoEmbedding[]> {
  const { data, error } = await supabase.rpc('get_knn_neighbors', {
    target_memo_name: memoName,
    k: limit,
  });

  if (error) {
    console.error('Error finding similar memos:', error);
    return [];
  }

  // Map the RPC result to MemoEmbedding format
  return (data || []).map(
    (d: { memo_name: string; content: string; similarity: number }) => ({
      memo_name: d.memo_name,
      content: d.content,
      embedding: [], // Not returned from this RPC
    })
  );
}
