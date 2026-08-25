export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface AuthorBrief {
  id: number
  nickname: string
  avatar_url: string
}

export interface TagBrief {
  id: number
  name: string
}

export interface Tag extends TagBrief {
  description?: string
  usage_count: number
}

export interface Category {
  id: number
  name: string
  slug: string
}

export interface ArticleSummary {
  id: number
  title: string
  summary: string
  tags: TagBrief[]
  author: AuthorBrief
  views: number
  likes_count: number
  favorites_count: number
  comments_count: number
  published_at: string | null
  pinned: boolean
}

export interface ArticleListResult {
  list: ArticleSummary[]
  total: number
  page: number
  page_size: number
}

export interface ArticleDetail extends ArticleSummary {
  content: string
  liked: boolean
  favorited: boolean
}

export interface CommentItem {
  id: number
  article_id: number
  author_id: number
  author: AuthorBrief
  content: string
  likes_count: number
  created_at: string
  replies: CommentItem[]
}

export interface UserProfile {
  id: number
  email: string
  nickname: string
  avatar_url: string
  bio: string
}

export interface LoginResult {
  access_token: string
  refresh_token: string
}

export interface ArticleListQuery {
  page?: number
  page_size?: number
  category_id?: number
  tag_id?: number
  keyword?: string
  author_id?: number
  sort?: string
}

export interface ArticleForm {
  title: string
  summary: string
  content: string
  status: 'draft' | 'published'
  tag_ids: number[]
  tag_names: string[]
}

export interface LikeResult {
  liked: boolean
  likes_count: number
}

export interface FavoriteResult {
  favorited: boolean
  favorites_count: number
}

export interface SkillSummary {
  id: number
  name: string
  description: string
  repo_url: string
  tags: TagBrief[]
  author: AuthorBrief
  views: number
  downloads: number
  likes_count: number
  favorites_count: number
  comments_count: number
  status: string
  published_at: string | null
}

export interface SkillDetail extends SkillSummary {
  zip_url: string
  zip_filename: string
  file_size: number
  liked: boolean
  favorited: boolean
}

export interface SkillListResult {
  list: SkillSummary[]
  total: number
  page: number
  page_size: number
}

export interface McpServerSummary {
  id: number
  name: string
  description: string
  repo_url: string
  tags: TagBrief[]
  author: AuthorBrief
  views: number
  downloads: number
  likes_count: number
  favorites_count: number
  comments_count: number
  status: string
  published_at: string | null
}

export interface McpServerDetail extends McpServerSummary {
  tools_json: string
  readme: string
  zip_url: string
  zip_filename: string
  file_size: number
  liked: boolean
  favorited: boolean
}

export interface McpServerListResult {
  list: McpServerSummary[]
  total: number
  page: number
  page_size: number
}

export interface ResourceCommentItem {
  id: number
  resource_id: number
  author_id: number
  author: AuthorBrief
  content: string
  likes_count: number
  created_at: string
  replies: ResourceCommentItem[]
}

export interface SearchResult {
  id: number
  type: 'article' | 'skill' | 'mcp_server'
  title: string
  summary: string
  views: number
  likes_count: number
  created_at: string
}

export interface SearchResponse {
  items: SearchResult[]
  total: number
  page: number
  page_size: number
  counts?: {
    article: number
    skill: number
    mcp_server: number
  }
}
