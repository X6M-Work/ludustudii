import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

export interface CreateLearningCardRequest {
  title: string
  description: string
}

export interface CreateLearningCardResponse {
  id: string
}

export interface LearningCard {
  id: string
  title: string
  description: string
}

export async function createLearningCard(
  payload: CreateLearningCardRequest,
): Promise<CreateLearningCardResponse> {
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

export async function getLearningCard(id: string): Promise<LearningCard> {
  const response = await fetch(`/api/lcards/${encodeURIComponent(id)}`)

  if (!response.ok) {
    throw new Error(`Failed to load learning card: ${response.statusText}`)
  }

  return response.json()
}

export function useLearningCard(id: string) {
  return useQuery({
    queryKey: ['lcards', id],
    queryFn: () => getLearningCard(id),
  })
}
