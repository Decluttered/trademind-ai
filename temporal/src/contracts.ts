export interface PublishListingInput {
  workspaceId: number;
  publicationId: string;
}

export interface PublicationActivities {
  revalidateListing(input: PublishListingInput): Promise<void>;
  publishListing(input: PublishListingInput): Promise<void>;
  reconcileListing(input: PublishListingInput): Promise<void>;
}
