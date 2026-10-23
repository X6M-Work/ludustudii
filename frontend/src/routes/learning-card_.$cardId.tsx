import { createFileRoute } from '@tanstack/react-router'
import { useLearningCard } from '@/api/learning-cards'
import { LearningCardForm } from '@/components/learning-card-form'
import { Card, CardContent } from '@/components/ui/card'
import { FieldError } from '@/components/ui/field'

export const Route = createFileRoute('/learning-card_/$cardId')({
  component: LearningCardEditRoute,
})

function LearningCardEditRoute() {
  const { cardId } = Route.useParams()
  const learningCard = useLearningCard(cardId)

  if (learningCard.isSuccess) {
    return <LearningCardForm key={learningCard.data.id} card={learningCard.data} />
  }

  return (
    <div className="mx-auto max-w-xl p-6">
      <Card>
        <CardContent>
          {learningCard.isPending ? (
            <span className="text-muted-foreground">Loading...</span>
          ) : (
            <FieldError>{learningCard.error.message}</FieldError>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
