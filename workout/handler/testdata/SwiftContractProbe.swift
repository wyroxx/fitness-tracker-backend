import Foundation

struct ContractFailure: Error { let message: String }

@main
struct SwiftContractProbe {
    static func check(_ condition: Bool, _ message: String) throws {
        if !condition { throw ContractFailure(message: message) }
    }

    static func main() async throws {
        let client = APIClient(baseURL: CommandLine.arguments[1], token: "contract-test")
        let groups = try await MuscleGroupRepositoryImpl(apiClient: client).getMuscleGroups()
        try check(groups.count == 1 && groups[0].name == "Chest", "muscle group decoding")
        let exercises = try await ExerciseRepositoryImpl(apiClient: client).getExercises(muscleGroupId: groups[0].id)
        try check(exercises.count == 1 && exercises[0].description == "Barbell press", "exercise decoding")
        let repository = WorkoutRepositoryImpl(apiClient: client)
        let empty = try await repository.getWorkouts()
        try check(empty.isEmpty, "empty history decoding")
        let exercise = exercises[0]
        let workout = Workout(
            title: "Swift contract test",
            startedAt: "2026-10-25T12:00:00Z",
            exercises: [
                WorkoutExercise(id: exercise.id, name: exercise.name, sets: [
                    WorkoutSet(setNumber: 1, weight: 52, reps: 10),
                    WorkoutSet(setNumber: 2, weight: 0, reps: 8),
                    WorkoutSet(setNumber: 3, weight: nil, reps: nil)
                ]),
                WorkoutExercise(id: exercise.id, name: exercise.name, sets: [])
            ]
        )
        try await repository.createWorkout(workout)
        let loaded = try await repository.getWorkouts()
        try check(loaded.count == 1, "saved training missing")
        let result = loaded[0]
        try check(result.title == workout.title && result.startedAt == workout.startedAt, "training metadata")
        try check(result.exercises.count == 2, "exercise count")
        let first = result.exercises[0]
        try check(first.id == exercise.id && first.name == exercise.name, "exercise identity")
        try check(first.sets.count == 3, "set count")
        try check(first.sets[0].weight == 52 && first.sets[0].reps == 10, "integer set data")
        try check(first.sets[1].weight == 0, "zero weight")
        try check(first.sets[2].weight == nil && first.sets[2].reps == nil, "optional set data")
        try check(result.exercises[1].sets.isEmpty, "empty sets decoding")
        print("Actual Swift APIClient and repositories: catalog, create and list passed")
    }
}
