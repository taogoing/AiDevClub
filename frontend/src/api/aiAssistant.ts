import http from './http'
import type { ApiResponse } from '@/types'

export interface AiAssistantRequest {
  question: string
  article_id?: number
}

export interface AiCitation {
  article_id?: number
  title?: string
  heading_path?: string | string[]
  chunk_id?: string
  text?: string
}

export interface AiAssistantAnswer {
  answer: string
  citations?: AiCitation[]
}

export function askAiAssistant(payload: AiAssistantRequest) {
  return http.post<ApiResponse<AiAssistantAnswer>>('/api/v1/ai/ask', payload)
}
