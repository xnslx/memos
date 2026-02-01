import { createClient } from "@supabase/supabase-js";

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL;
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY;

if (!supabaseUrl || !supabaseAnonKey) {
  throw new Error("Missing Supabase environment variables. Check your .env file.");
}

export const supabase = createClient(supabaseUrl, supabaseAnonKey);

export interface MemoEmbedding {
  id?: string;
  memo_name: string;
  content: string;
  embedding: number[];
  intent_summary?: string | null;
  intent_embedding?: number[] | null;
  chunk_count?: number;
  created_at?: string;
  updated_at?: string;
}
