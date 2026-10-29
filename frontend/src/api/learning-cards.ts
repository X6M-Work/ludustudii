import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

export interface CreateLearningCardRequest {
  title: string
  description: string
}

export interface CreateLearningCardResponse {
  id: string
}

export interface UpdateLearningCardRequest {
  id: string
  title: string
  description: string
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

export async function updateLearningCard({
  id,
  title,
  description,
}: UpdateLearningCardRequest): Promise<void> {
  const response = await fetch(`/api/lcards/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ title, description }),
  })

  if (!response.ok) {
    throw new Error(`Failed to update learning card: ${response.statusText}`)
  }
}

export function useUpdateLearningCard() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: updateLearningCard,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['lcards'] })
    },
  })
}

// Expected delete endpoint contract: DELETE /api/lcards/:id returns no content.
export async function deleteLearningCard(id: string): Promise<void> {
  const response = await fetch(`/api/lcards/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })

  if (!response.ok) {
    throw new Error(`Failed to delete learning card: ${response.statusText}`)
  }
}

export function useDeleteLearningCard() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteLearningCard,
    onSuccess: (_data, id) => {
      // Drop the deleted card's cache entry so it is not refetched into a 404.
      queryClient.removeQueries({ queryKey: ['lcards', id], exact: true })
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

// Expected list endpoint contract: GET /api/lcards returns
// { items: LearningCard[], next_cursor: string | null }.
// Only the first page is loaded; next_cursor is not followed yet.
export async function getLearningCards(signal?: AbortSignal): Promise<LearningCard[]> {
  const response = await fetch('/api/lcards', { signal })

  if (!response.ok) {
    throw new Error(`Failed to load learning cards: ${response.statusText}`)
  }

  const page: unknown = await response.json()
  if (
    typeof page !== 'object' ||
    page === null ||
    !('items' in page) ||
    !Array.isArray(page.items)
  ) {
    throw new Error('Expected a page of learning cards')
  }

  return page.items
}

export function useLearningCards() {
  return useQuery({
    queryKey: ['lcards', 'list'],
    queryFn: ({ signal }) => getLearningCards(signal),
  })
}
