import Markdown, { type Components } from 'react-markdown'
import { useLearningCard, type LearningCard } from '@/api/learning-cards'
import {
  Card,
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
  return (
    <div className="mx-auto max-w-xl p-6">
      <Card>
        <CardHeader>
          <CardTitle className="wrap-break-word">
            <Markdown components={markdownComponents}>{card.title}</Markdown>
          </CardTitle>
        </CardHeader>
        <Separator />
        <CardContent className="wrap-break-word">
          <Markdown components={markdownComponents}>{card.description}</Markdown>
        </CardContent>
      </Card>
    </div>
  )
}
