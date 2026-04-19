export type Student = {
  student_id: string
  name: string
  email?: string
  class?: string
  grade?: number
  current_rating: number
  max_rating: number
  match_count: number
  redeem_points: number
  tier_name?: string
  tier_color?: string
  rank?: number
}

export type ContestRewardType = "none" | "foundation_exam" | "brand_monthly" | "brand_final"

export type Contest = {
  id: number
  name: string
  date: string
  weight: number
  reward_type: ContestRewardType
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

export type PointCategory = "contest_award" | "progress_award" | "organizer_reward"

export type PointSource = "auto" | "manual"

export type PointRecord = {
  id: number
  student_id: string
  contest_id?: number
  category: PointCategory
  source: PointSource
  points: number
  title: string
  description?: string
  created_at: string
  student?: Pick<Student, "student_id" | "name">
  contest?: Pick<Contest, "id" | "name" | "date">
}
