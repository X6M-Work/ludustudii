import { createFileRoute } from '@tanstack/react-router'
import { LearningCardsOverview } from '@/components/learning-cards-overview'

export const Route = createFileRoute('/learning-cards')({
  component: LearningCardsOverview,
})
