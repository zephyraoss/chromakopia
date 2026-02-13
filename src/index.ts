import { Elysia, t } from "elysia";
import { queue } from "./queue";
import { runFpcalcOnUpload, runFpcalcOnUrl } from "./fingerprint";
import { identifyWithAcoustID, AcoustIDResponse } from "./acoustid";

interface CleanRecording {
  title: string;
  artist: string;
  album?: string;
  duration: number;
}

interface CleanResult {
  id: string;
  score: number;
  recording: CleanRecording;
}

function cleanResponse(response: AcoustIDResponse) {
  if (!response.results || response.results.length === 0) {
    return { status: "ok", matches: [] as CleanResult[] };
  }

  const matches: CleanResult[] = [];

  for (const result of response.results) {
    if (!result.recordings || result.recordings.length === 0) continue;

    const recording = result.recordings[0];
    const artist = recording.artists?.[0]?.name || "Unknown Artist";
    
    let album: string | undefined;
    if (recording.releasegroups && recording.releasegroups.length > 0) {
      album = recording.releasegroups[0].title;
    }

    matches.push({
      id: result.id,
      score: Math.round(result.score * 100) / 100,
      recording: {
        title: recording.title || "Unknown Title",
        artist,
        album,
        duration: Math.round(recording.duration || 0),
      },
    });
  }

  return { status: "ok", matches };
}

const app = new Elysia()
  .post(
    "/identify/file",
    async ({ body }) => {
      const { file } = body as { file: File };

      if (!file || !file.name) {
        return { status: "ERROR", error: "No file provided" };
      }

      try {
        const acoustidResult = await queue.enqueue(async () => {
          const fpcalcResult = await runFpcalcOnUpload(file, file.name);
          return identifyWithAcoustID(
            fpcalcResult.fingerprint,
            fpcalcResult.duration
          );
        });

        return cleanResponse(acoustidResult);
      } catch (error) {
        return {
          status: "ERROR",
          error: error instanceof Error ? error.message : "Unknown error",
        };
      }
    },
    {
      body: t.Object({
        file: t.File(),
      }),
    }
  )
  .post(
    "/identify/url",
    async ({ body }) => {
      const { url } = body as { url: string };

      if (!url) {
        return { status: "ERROR", error: "No URL provided" };
      }

      try {
        const acoustidResult = await queue.enqueue(async () => {
          const fpcalcResult = await runFpcalcOnUrl(url);
          return identifyWithAcoustID(
            fpcalcResult.fingerprint,
            fpcalcResult.duration
          );
        });

        return cleanResponse(acoustidResult);
      } catch (error) {
        return {
          status: "ERROR",
          error: error instanceof Error ? error.message : "Unknown error",
        };
      }
    },
    {
      body: t.Object({
        url: t.String(),
      }),
    }
  )
  .get("/health", () => {
    return { status: "ok", queueLength: queue.length };
  })
  .listen(3000);

console.log(`Server running at http://localhost:${app.server?.port}`);

export type App = typeof app;
