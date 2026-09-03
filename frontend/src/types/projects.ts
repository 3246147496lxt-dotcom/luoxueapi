export type ProjectMemoryMode = 'default' | 'project-only'

export interface ProjectFile {
  id: string
  name: string
  size?: number
  mimeType?: string
  createdAt?: string
}

/** A light-weight conversation projection returned by the project endpoint. */
export interface ProjectConversation {
  id: string
  title: string
  model?: string
  messageCount?: number
  createdAt?: string
  updatedAt?: string
}

export interface Project {
  id: string
  name: string
  icon: string
  color: string
  memoryMode: ProjectMemoryMode
  instructions: string
  conversationIds: string[]
  conversations: ProjectConversation[]
  files: ProjectFile[]
  createdAt: number
  updatedAt: number
  /** True when the project only exists in the offline browser cache. */
  localOnly?: boolean
}

export interface ProjectInput {
  name: string
  icon?: string
  color?: string
  memoryMode?: ProjectMemoryMode
  instructions?: string
}
