import { useNavigate } from '@tanstack/react-router'
import { useForm } from '@tanstack/react-form'
import {
  useCreateLearningCard,
  useUpdateLearningCard,
  type LearningCard,
} from '@/api/learning-cards'
import { LearningCardDisplay } from '@/components/learning-card-view'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Textarea } from '@/components/ui/textarea'

const TITLE_MAX_LENGTH = 200
const DESCRIPTION_MAX_LENGTH = 2000

// The maxLength attribute counts UTF-16 code units of the raw value, so it
// only caps runaway pastes; validation reports the real limit.
const INPUT_SLACK = 2

// The server stores fields in this form and checks their length after
// applying it. JS trim() also strips U+FEFF and keeps U+0085, unlike Go's
// strings.TrimSpace; at worst the server rejects what the form accepted.
const canonicalize = (value: string) => value.normalize('NFC').trim()

// Counts code points, as the server does, rather than UTF-16 code units.
const canonicalLength = (value: string) => [...canonicalize(value)].length

const validateTitle = ({ value }: { value: string }) => {
  const length = canonicalLength(value)
  if (length === 0) {
    return { message: 'Title is required' }
  }
  if (length > TITLE_MAX_LENGTH) {
    return { message: `Title must be at most ${TITLE_MAX_LENGTH} characters` }
  }
  return undefined
}

const validateDescription = ({ value }: { value: string }) => {
  const length = canonicalLength(value)
  if (length === 0) {
    return { message: 'Description is required' }
  }
  if (length > DESCRIPTION_MAX_LENGTH) {
    return {
      message: `Description must be at most ${DESCRIPTION_MAX_LENGTH} characters`,
    }
  }
  return undefined
}

interface LearningCardFormProps {
  card?: LearningCard
}

export function LearningCardForm({ card }: LearningCardFormProps) {
  const navigate = useNavigate()
  const createLearningCard = useCreateLearningCard()
  const updateLearningCard = useUpdateLearningCard()
  const isEditing = Boolean(card)
  const mutation = isEditing ? updateLearningCard : createLearningCard

  const form = useForm({
    defaultValues: {
      title: card?.title ?? '',
      description: card?.description ?? '',
    },
    onSubmit: async ({ value, formApi }) => {
      const input = {
        title: canonicalize(value.title),
        description: canonicalize(value.description),
      }
      if (card) {
        await updateLearningCard.mutateAsync({
          id: card.id,
          ...input,
        })
        await navigate({ to: '/learning-cards' })
      } else {
        await createLearningCard.mutateAsync(input)
        formApi.reset()
      }
    },
  })

  return (
    <div className="mx-auto grid max-w-6xl grid-cols-1 items-start gap-6 p-6 md:grid-cols-2">
      <Card className="min-w-0">
        <CardHeader>
          <CardTitle>{isEditing ? 'Update Card' : 'Add Card'}</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              e.stopPropagation()
              form.handleSubmit()
            }}
          >
            <FieldGroup>
              <form.Field
                name="title"
                validators={{
                  onChange: validateTitle,
                  onSubmit: validateTitle,
                }}
              >
                {(field) => {
                  const isInvalid = field.state.meta.errors.length > 0
                  return (
                    <Field data-invalid={isInvalid}>
                      <div className="flex items-center justify-between">
                        <FieldLabel htmlFor={field.name}>Title</FieldLabel>
                        <FieldDescription>
                          {canonicalLength(field.state.value)}/{TITLE_MAX_LENGTH}
                        </FieldDescription>
                      </div>
                      <Textarea
                        className="min-h-5"
                        id={field.name}
                        name={field.name}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        maxLength={TITLE_MAX_LENGTH * INPUT_SLACK}
                        aria-invalid={isInvalid}
                        placeholder="Enter title..."
                      />
                      <div className="min-h-4">
                        {isInvalid && (
                          <FieldError errors={field.state.meta.errors} />
                        )}
                      </div>
                    </Field>
                  )
                }}
              </form.Field>

              <form.Field
                name="description"
                validators={{
                  onChange: validateDescription,
                  onSubmit: validateDescription,
                }}
              >
                {(field) => {
                  const isInvalid = field.state.meta.errors.length > 0
                  return (
                    <Field data-invalid={isInvalid}>
                      <div className="flex items-center justify-between">
                        <FieldLabel htmlFor={field.name}>Description</FieldLabel>
                        <FieldDescription>
                          {canonicalLength(field.state.value)}/{DESCRIPTION_MAX_LENGTH}
                        </FieldDescription>
                      </div>
                      <Textarea
                        className="min-h-50"
                        id={field.name}
                        placeholder="Enter description..."
                        name={field.name}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        maxLength={DESCRIPTION_MAX_LENGTH * INPUT_SLACK}
                        aria-invalid={isInvalid}
                      />
                      <div className="min-h-4">
                        {isInvalid && (
                          <FieldError errors={field.state.meta.errors} />
                        )}
                      </div>
                    </Field>
                  )
                }}
              </form.Field>

              {mutation.isError && (
                <FieldError>
                  {mutation.error.message}
                </FieldError>
              )}

              <form.Subscribe
                selector={(state) => [state.canSubmit, state.isSubmitting]}
              >
                {([canSubmit, isSubmitting]) => (
                  <Button
                    type="submit"
                    disabled={
                      !canSubmit || isSubmitting || mutation.isPending
                    }
                  >
                    {isSubmitting || mutation.isPending
                      ? isEditing
                        ? 'Updating...'
                        : 'Creating...'
                      : isEditing
                        ? 'Update Card'
                        : 'Create Card'}
                  </Button>
                )}
              </form.Subscribe>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <form.Subscribe selector={(state) => state.values}>
        {(values) => <LearningCardDisplay card={values} />}
      </form.Subscribe>
    </div>
  )
}

