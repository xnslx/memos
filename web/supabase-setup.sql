-- Enable the pgvector extension
create extension if not exists vector;

-- Create the memo_embeddings table
create table if not exists memo_embeddings (
  id uuid default gen_random_uuid() primary key,
  memo_name text unique not null,
  content text not null,
  embedding vector(1536), -- OpenAI text-embedding-3-small outputs 1536 dimensions
  created_at timestamp with time zone default timezone('utc'::text, now()) not null,
  updated_at timestamp with time zone default timezone('utc'::text, now()) not null
);


-- Create an index for faster vector similarity searches
create index if not exists memo_embeddings_embedding_idx
  on memo_embeddings
  using ivfflat (embedding vector_cosine_ops)
  with (lists = 100);

-- Create a function to search for similar memos
create or replace function match_memos (
  query_embedding vector(1536),
  match_threshold float,
  match_count int
)
returns table (
  id uuid,
  memo_name text,
  content text,
  embedding vector(1536),
  created_at timestamp with time zone,
  updated_at timestamp with time zone,
  similarity float
)
language sql stable
as $$
  select
    memo_embeddings.id,
    memo_embeddings.memo_name,
    memo_embeddings.content,
    memo_embeddings.embedding,
    memo_embeddings.created_at,
    memo_embeddings.updated_at,
    1 - (memo_embeddings.embedding <=> query_embedding) as similarity
  from memo_embeddings
  where 1 - (memo_embeddings.embedding <=> query_embedding) > match_threshold
  order by memo_embeddings.embedding <=> query_embedding
  limit match_count;
$$;


-- Enable Row Level Security (RLS) - optional but recommended
alter table memo_embeddings enable row level security;

-- Create a policy to allow all operations for now (adjust based on your auth needs)
create policy "Allow all operations" on memo_embeddings
  for all
  using (true)
  with check (true);
  

