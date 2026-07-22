import * as ImagePicker from 'expo-image-picker';

import { apiService } from './api';

const EXTENSION_CONTENT_TYPES: Record<string, string> = {
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
};

function contentTypeOf(uri: string): string {
  const extension = uri.split('?')[0].split('.').pop()?.toLowerCase() ?? '';
  return EXTENSION_CONTENT_TYPES[extension] ?? 'image/jpeg';
}

/** Opens the library and returns the chosen image's local URI, or null. */
export async function pickImage(): Promise<string | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return null;
  }

  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ImagePicker.MediaTypeOptions.Images,
    quality: 0.7,
    allowsEditing: true,
  });

  return result.canceled ? null : result.assets[0].uri;
}

/**
 * Uploads a local image straight to S3 with a presigned URL and returns its
 * public URL. The bytes never pass through our backend.
 */
export async function uploadImage(localUri: string): Promise<string> {
  const contentType = contentTypeOf(localUri);
  const filename = localUri.split('/').pop() || 'image.jpg';

  const { upload_url, public_url } = await apiService.presignUpload(filename, contentType);

  // fetch() streams the file:// body straight to S3; no base64 round trip.
  const file = await fetch(localUri).then((response) => response.blob());
  const uploaded = await fetch(upload_url, {
    method: 'PUT',
    headers: { 'Content-Type': contentType },
    body: file,
  });

  if (!uploaded.ok) {
    throw new Error(`Upload failed (${uploaded.status})`);
  }

  return public_url;
}
