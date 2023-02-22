package cmdchain

import (
	"context"
	"io"
	"os/exec"
)

// ChainBuilder contains methods for joining new commands to the current cain or finalize them.
type ChainBuilder interface {
	// Join create a new command by the given name and the given arguments. This command then will join
	// the chain. If there is a command which joined before, their stdout/stderr will redirected to this
	// command in stdin (depending of its configuration). After calling Join the command can be more
	// configured. After calling another Join this command can not be configured again. Instead the
	// configuration of the next command will begin.
	Join(name string, args ...string) CommandBuilder

	// JoinCmd takes the given command and join them to the chain. If there is a command which joined
	// before, their stdout/stderr will redirected to this command in stdin (depending of its configuration).
	// Therefore the input (stdin) and output (stdout/stderr) will be manipulated by the chain building process.
	// The streams must not be configured outside the chain builder. Otherwise the chain building process will
	// be failed after Run will be called. After calling JoinCmd the command can be more configured. After
	// calling another Join this command can not be configured again. Instead the configuration of the
	// next command will begin.
	JoinCmd(cmd *exec.Cmd) CommandBuilder

	// JoinWithContext is like Join but includes a context to the created command. The provided context is used
	// to kill the process (by calling os.Process.Kill) if the context becomes done before the command completes
	// on its own.
	JoinWithContext(ctx context.Context, name string, args ...string) CommandBuilder

	// Finalize will finish the command joining process. After calling this method no command can be joined anymore.
	// Instead final configurations can be made and the chain is ready to run.
	Finalize() FinalizedBuilder
}

// FirstCommandBuilder contains methods for building the chain. Especially it contains configuration which can be
// made only for the first command in the chain.
type FirstCommandBuilder interface {
	ChainBuilder

	// WithInput configures the input stream(s) for the first command in the chain. If multiple streams are
	// configured, this streams will read in parallel (not sequential!). So be aware of concurrency issues.
	// If this behavior is not wanted, me the io.MultiReader is a better choice.
	WithInput(sources ...io.Reader) ChainBuilder
}

// CommandApplier is a function which will get the command's index and the command's reference
type CommandApplier func(index int, command *exec.Cmd)

// CommandBuilder contains methods for configuring the previous joined command.
type CommandBuilder interface {
	ChainBuilder

	// Apply will call the given CommandApplier with the previously joined command. The CommandApplier can do anything
	// with the previously joined command. The CommandApplier will be called directly so the command which the applier
	// will be received has included all changes which made before this function call.
	// ATTENTION: Be aware of the changes the CommandApplier will make. This can clash with the changes the building
	// pipeline will make!
	Apply(CommandApplier) CommandBuilder

	// ApplyBeforeStart will call the given CommandApplier with the previously joined command. The CommandApplier can do
	// anything with the previously joined command. The CommandApplier will be called before the command will be started
	// so the command is almost finished (all streams are configured and so on).
	// ATTENTION: Be aware of the changes the CommandApplier will make. This can clash with the changes the building
	// pipeline will make!
	ApplyBeforeStart(CommandApplier) CommandBuilder

	// ForwardError will configure the previously joined command to redirect all its stderr output to the next
	// command's input. If WithErrorForks is also used, the stderr output of the previously joined command will
	// be redirected to both: stdin of the next command AND the configured fork(s).
	// If ForwardError is not used, the stderr output of the previously joined command will be dropped. But if
	// WithErrorForks is used, the stderr output will be redirected to the configured fork(s).
	ForwardError() CommandBuilder

	// DiscardStdOut will configure the previously joined command to drop all its stdout output. So the stdout does NOT
	// redirect to the next command's stdin. If WithOutputForks is also used, the output of the previously joined
	// command will be redirected to this fork(s). It will cause an invalid stream configuration error if the stderr is
	// also discarded (which is the default case)! So it should be used in combination of ForwardError.
	DiscardStdOut() CommandBuilder

	// WithOutputForks will configure the previously joined command to redirect their stdout output to the configured
	// target(s). The configured writer will be written in parallel so streaming is possible. If the previously
	// joined command is also configured to redirect its stdout to the next command's input, the stdout output will
	// redirected to both: stdin of the next command AND the configured fork(s).
	// ATTENTION: If one of the given writer will be closed before the command ends the command will be exited. This is
	// because of the this method uses the io.MultiWriter. And it will close the writer if on of them is closed.
	WithOutputForks(targets ...io.Writer) CommandBuilder

	// WithAdditionalOutputForks is similar to WithOutputForks except that the given targets will be added to the
	// command and not be overwritten.
	WithAdditionalOutputForks(targets ...io.Writer) CommandBuilder

	// WithErrorForks will configure the previously joined command to redirect their stderr output to the configured
	// target(s). The configured writer will be written in parallel so streaming is possible. If the previously
	// joined command is also configured to redirect its stderr to the next command's input, the stderr output will
	// redirected to both: stdin of the next command AND the configured fork(s).
	// ATTENTION: If one of the given writer will be closed before the command ends the command will be exited. This is
	// because of the this method uses the io.MultiWriter. And it will close the writer if on of them is closed.
	WithErrorForks(targets ...io.Writer) CommandBuilder

	// WithAdditionalErrorForks is similar to WithErrorForks except that the given targets will be added to the
	// command and not be overwritten.
	WithAdditionalErrorForks(targets ...io.Writer) CommandBuilder

	// WithInjections will configure the previously joined command to read from the given sources AND the predecessor
	// command's stdout or stderr (depending on the configuration). This streams (stdout/stderr of predecessor command
	// and the given sources) will read in parallel (not sequential!). So be aware of concurrency issues.
	// If this behavior is not wanted, me the io.MultiReader is a better choice.
	WithInjections(sources ...io.Reader) CommandBuilder

	// WithEnvironment will configure the previously joined command to use the given environment variables. Key-value
	// pair(s) must be passed as arguments. Where the first represents the key and the second the value of the
	// environment variable.
	WithEnvironment(envMap ...interface{}) CommandBuilder

	// WithEnvironmentMap will configure the previously joined command to use the given environment variables.
	WithEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder

	// WithAdditionalEnvironment will do almost the same thing as WithEnvironment expecting that the given key-value
	// pairs will be joined with the environment variables of the current process.
	WithAdditionalEnvironment(envMap ...interface{}) CommandBuilder

	// WithAdditionalEnvironmentMap will do almost the same thing as WithEnvironmentMap expecting that the given
	// values will be joined with the environment variables of the current process.
	WithAdditionalEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder

	// WithWorkingDirectory will configure the previously joined command to use the specifies the working directory.
	// Without setting the working directory, the calling process's current directory will be used.
	WithWorkingDirectory(workingDir string) CommandBuilder

	// WithErrorChecker will configure the previously joined command to use the given error checker. In some cases
	// the commands will return a non-zero exit code, which will normally cause an error at the FinalizedBuilder.Run().
	// To avoid that you can use a ErrorChecker to ignore these kind of errors. There exists a set of functions which
	// create a such ErrorChecker: IgnoreExitCode, IgnoreExitErrors, IgnoreAll, IgnoreNothing
	WithErrorChecker(ErrorChecker) CommandBuilder
}

// FinalizedBuilder contains methods for configuration the the finalized chain. At this step the chain can be running.
type FinalizedBuilder interface {

	// WithOutput configures the stdout stream(s) for the last command in the chain. If there is more than one target
	// given io.MultiWriter will be used as command's stdout. So in that case if there was one of the given targets
	// closed before the chain normally ends, the chain will be exited. This is because of the behavior of the
	// io.MultiWriter.
	WithOutput(targets ...io.Writer) FinalizedBuilder

	// WithAdditionalOutput is similar to WithOutput except that the given targets will be added to the
	// command and not be overwritten.
	WithAdditionalOutput(targets ...io.Writer) FinalizedBuilder

	// WithError configures the stderr stream(s) for the last command in the chain. If there is more than one target
	// given io.MultiWriter will be used as command's stdout. So in that case if there was one of the given targets
	// closed before the chain normally ends, the chain will be exited. This is because of the behavior of the
	// io.MultiWriter.
	WithError(targets ...io.Writer) FinalizedBuilder

	// WithAdditionalError is similar to WithError except that the given targets will be added to the
	// command and not be overwritten.
	WithAdditionalError(targets ...io.Writer) FinalizedBuilder

	// WithGlobalErrorChecker will configure the complete chain to use the given error checker. If there is an error
	// checker configured for a special command, this error checker will be skipped for these one. In some cases
	// the commands will return a non-zero exit code, which will normally cause an error at the Run().
	// To avoid that you can use a ErrorChecker to ignore these kind of errors. There exists a set of functions which
	// create a such ErrorChecker: IgnoreExitCode, IgnoreExitErrors, IgnoreAll, IgnoreNothing
	WithGlobalErrorChecker(ErrorChecker) FinalizedBuilder

	// Run will execute the command chain. It will start all underlying commands and wait after completion of all of
	// them. If the building of the chain was failed, an error will returned before the commands are started! In that
	// case an MultipleErrors will be returned. If any command starting failed, the run will the error (single) of
	// starting. All previously started commands should be exited in that case. Following commands will not be started.
	// If any error occurs while commands are running, a MultipleErrors will return within all errors per
	// command.
	Run() error

	// RunAndGet works like Run in addition the function will return the stdout and stderr of the command chain. Be
	// careful with this convenience function because the stdout and stderr will be stored in memory!
	RunAndGet() (string, string, error)

	// String returns a string representation of the command chain.
	String() string
}
