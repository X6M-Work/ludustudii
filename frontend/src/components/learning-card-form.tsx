import { useNavigate } from '@tanstack/react-router'
import { useForm } from '@tanstack/react-form'
import {
  useCreateLearningCard,
  useUpdateLearningCard,
  type LearningCard,
} from '@/api/learning-cards'
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

const validateTitle = ({ value }: { value: string }) => {
  if (!value.trim()) {
    return { message: 'Title is required' }
  }
  if (value.length > 200) {
    return { message: 'Title must be at most 200 characters' }
  }
  return undefined
}

const validateDescription = ({ value }: { value: string }) => {
  if (!value.trim()) {
    return { message: 'Description is required' }
  }
  if (value.length > 2000) {
    return { message: 'Description must be at most 2000 characters' }
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
      if (card) {
        await updateLearningCard.mutateAsync({
          id: card.id,
          ...value,
        })
        await navigate({ to: '/learning-cards' })
      } else {
        await createLearningCard.mutateAsync(value)
        formApi.reset()
      }
    },
  })

  return (
    <div className="mx-auto max-w-xl p-6">
      <Card>
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
                          {field.state.value.length}/200
                        </FieldDescription>
                      </div>
                      <Textarea
                        className="min-h-5"
                        id={field.name}
                        name={field.name}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        maxLength={200}
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
                          {field.state.value.length}/2000
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
                        maxLength={2000}
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
    </div>
  )
}

