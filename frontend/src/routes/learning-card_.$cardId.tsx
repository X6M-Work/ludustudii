import { createFileRoute } from '@tanstack/react-router'
import { LearningCardView } from '@/components/learning-card-view'

export const Route = createFileRoute('/learning-card_/$cardId')({
  component: LearningCardViewRoute,
})

function LearningCardViewRoute() {
  const { cardId } = Route.useParams()
  return <LearningCardView cardId={cardId} />
}
