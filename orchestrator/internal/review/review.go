package review

import (
	"fmt"
	"time"

	"orchestrator/internal/context"
	"orchestrator/internal/handoff"
	"orchestrator/internal/session"
	"orchestrator/internal/verify"
)

type Result struct {
	ReturnCode int
	Stdout     string
	Stderr     string
}

func buildReviewPrompt(reviewType string, projectName string, milestones []context.MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) (string, string) {
	bundle := context.BuildBundle(milestones, handoffNotes, verifyResults)

	if context.ExceedsThreshold(bundle, 0) {
		degraded := false
		if len(verifyResults) > 0 {
			bundle = context.Degrade(bundle, context.DegradeNoVerify)
			degraded = true
		}
		if len(handoffNotes) > 0 && context.ExceedsThreshold(bundle, 0) {
			bundle = context.Degrade(bundle, context.DegradeNoHandoff)
			degraded = true
		}
		if !degraded {
			bundle = context.Degrade(bundle, context.DegradeMinimal)
		}
	}

	target := projectName
	prompt := fmt.Sprintf(
		"Review: %s for %s\n\n## Context Bundle\n\n%s\n\nPlease perform a %s review and provide feedback. Write your review to HANDOFF.md in the current directory.",
		reviewType, target, bundle, reviewType,
	)
	return bundle, prompt
}

func Phase(milestoneName, milestoneSpec string, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	ms := []context.MilestoneInfo{{Name: milestoneName, Spec: milestoneSpec}}
	_, prompt := buildReviewPrompt("phase", milestoneName, ms, handoffNotes, verifyResults)
	result, err := session.Run(prompt, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}

func Final(projectName string, milestones []context.MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	_, prompt := buildReviewPrompt("final", projectName, milestones, handoffNotes, verifyResults)
	result, err := session.Run(prompt, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}
