import { createFileRoute } from '@tanstack/react-router'
import { LearningCardForm } from '@/components/learning-card-form'

export const Route = createFileRoute('/learning-card')({
  component: LearningCardRoute,
})

function LearningCardRoute() {
  return <LearningCardForm />
}
