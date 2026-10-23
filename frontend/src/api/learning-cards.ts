import { useMutation, useQueryClient } from '@tanstack/react-query'

export interface CreateLearningCardRequest {
  title: string
  description: string
}

export interface LearningCard {
  id: string
}

export async function createLearningCard(
  payload: CreateLearningCardRequest,
): Promise<LearningCard> {
  const response = await fetch('/api/lcards', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    throw new Error(`Failed to create learning card: ${response.statusText}`)
  }

  return response.json()
}

export function useCreateLearningCard() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createLearningCard,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['lcards'] })
    },
  })
}

