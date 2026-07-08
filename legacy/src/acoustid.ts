const ACOUSTID_API_KEY = process.env.ACOUSTID_API_KEY ?? (() => {
  throw new Error("ACOUSTID_API_KEY not set in environment");
})();
const ACOUSTID_API_URL = "https://api.acoustid.org/v2/lookup";

export interface AcoustIDRecording {
  id: string;
  title: string;
  duration?: number;
  artists?: { id: string; name: string }[];
  releasegroups?: {
    id: string;
    title: string;
    type?: string;
  }[];
}

export interface AcoustIDResult {
  id: string;
  score: number;
  recordings?: AcoustIDRecording[];
}

export interface AcoustIDResponse {
  status: string;
  results: AcoustIDResult[];
}

export async function identifyWithAcoustID(
  fingerprint: string,
  duration: number
): Promise<AcoustIDResponse> {
  const formData = new URLSearchParams();
  formData.append("client", ACOUSTID_API_KEY);
  formData.append("duration", duration.toString());
  formData.append("fingerprint", fingerprint);
  formData.append("meta", "recordings releasegroups compress");

  const response = await fetch(ACOUSTID_API_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: formData.toString(),
  });

  if (!response.ok) {
    throw new Error(`AcoustID API error: ${response.status} ${response.statusText}`);
  }

  const data = await response.json() as AcoustIDResponse;
  return data;
}
