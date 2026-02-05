export type Student = {
  student_id: string
  name: string
  email?: string
  class?: string
  grade?: number
  current_rating: number
  max_rating: number
  match_count: number
}

export type Contest = {
  id: number
  name: string
  date: string
  weight: number
  participant_count: number
}

export type Result = {
  id: number
  student_id: string
  contest_id: number
  rank: number
  solved: number
  performance: number
  rating_before: number
  rating_after: number
  delta: number
  contest?: Contest
}

export type TierConfig = {
  id?: number
  name: string
  min_rating: number
  max_rating: number
  color: string
  order: number
}
