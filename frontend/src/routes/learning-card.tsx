import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/learning-card')({
  component: LearningCardForm,
})

function LearningCardForm() {
  return <h1>Add Learning Card</h1>
}