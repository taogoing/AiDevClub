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
  category_id: number
  category_name: string
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
  category_id: number
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
