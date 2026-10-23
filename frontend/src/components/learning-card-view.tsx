import { useState, type ReactNode } from 'react'
import Markdown, { type Components } from 'react-markdown'
import { useNavigate } from '@tanstack/react-router'
import {
  useDeleteLearningCard,
  useLearningCard,
  type LearningCard,
} from '@/api/learning-cards'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { FieldError } from '@/components/ui/field'
import { Separator } from '@/components/ui/separator'

// Tailwind preflight resets element styles, so markdown elements are styled here
const markdownComponents: Components = {
  h1: ({ node: _node, ...props }) => <h1 className="mt-4 mb-2 text-lg font-semibold first:mt-0" {...props} />,
  h2: ({ node: _node, ...props }) => <h2 className="mt-4 mb-2 text-base font-semibold first:mt-0" {...props} />,
  h3: ({ node: _node, ...props }) => <h3 className="mt-3 mb-1 text-sm font-semibold first:mt-0" {...props} />,
  h4: ({ node: _node, ...props }) => <h4 className="mt-3 mb-1 font-semibold first:mt-0" {...props} />,
  p: ({ node: _node, ...props }) => <p className="my-2 first:mt-0 last:mb-0" {...props} />,
  a: ({ node: _node, ...props }) => <a className="text-primary underline underline-offset-2" {...props} />,
  ul: ({ node: _node, ...props }) => <ul className="my-2 list-disc pl-5" {...props} />,
  ol: ({ node: _node, ...props }) => <ol className="my-2 list-decimal pl-5" {...props} />,
  li: ({ node: _node, ...props }) => <li className="my-0.5" {...props} />,
  blockquote: ({ node: _node, ...props }) => (
    <blockquote className="my-2 border-l-2 pl-3 text-muted-foreground" {...props} />
  ),
  code: ({ node: _node, ...props }) => <code className="bg-muted px-1 py-0.5 font-mono" {...props} />,
  pre: ({ node: _node, ...props }) => (
    <pre className="my-2 overflow-x-auto bg-muted p-3 font-mono [&>code]:p-0" {...props} />
  ),
  hr: () => <Separator className="my-4" />,
}

type LearningCardViewProps =
  | { cardId: string; card?: never }
  | { card: LearningCard; cardId?: never }

export function LearningCardView(props: LearningCardViewProps) {
  if (props.card) {
    return <LearningCardContent card={props.card} />
  }

  return <LearningCardById cardId={props.cardId} />
}

function LearningCardById({ cardId }: { cardId: string }) {
  const learningCard = useLearningCard(cardId)

  if (learningCard.isSuccess) {
    return <LearningCardContent card={learningCard.data} />
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

function LearningCardContent({ card }: { card: LearningCard }) {
  const navigate = useNavigate()

  return (
    <div className="mx-auto max-w-xl p-6">
      <LearningCardDisplay
        card={card}
        action={
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                navigate({
                  to: '/learning-card/$cardId',
                  params: { cardId: card.id },
                })
              }
            >
              Update
            </Button>
            <DeleteLearningCardButton cardId={card.id} />
          </div>
        }
      />
    </div>
  )
}

function DeleteLearningCardButton({ cardId }: { cardId: string }) {
  const [open, setOpen] = useState(false)
  const deleteCard = useDeleteLearningCard()

  return (
    <AlertDialog
      open={open}
      onOpenChange={(nextOpen) => {
        // Keep the dialog open until the request settles
        if (deleteCard.isPending) return
        if (!nextOpen) deleteCard.reset()
        setOpen(nextOpen)
      }}
    >
      <AlertDialogTrigger render={<Button variant="destructive" size="sm" />}>
        Delete
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this learning card?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. The card will be permanently removed.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {deleteCard.isError && <FieldError>{deleteCard.error.message}</FieldError>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleteCard.isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleteCard.isPending}
            onClick={() =>
              deleteCard.mutate(cardId, {
                onSuccess: () => setOpen(false),
              })
            }
          >
            {deleteCard.isPending ? 'Deleting...' : 'Delete'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

export function LearningCardDisplay({
  card,
  action,
}: {
  card: Pick<LearningCard, 'title' | 'description'>
  action?: ReactNode
}) {
  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="min-w-0 wrap-break-word">
          <Markdown components={markdownComponents}>{card.title}</Markdown>
        </CardTitle>
        {action && <CardAction>{action}</CardAction>}
      </CardHeader>
      <Separator />
      <CardContent className="min-w-0 wrap-break-word">
        <Markdown components={markdownComponents}>{card.description}</Markdown>
      </CardContent>
    </Card>
  )
}
