package review

import (
	"fmt"
	"time"

	"orc/internal/context"
	"orc/internal/handoff"
	"orc/internal/session"
	"orc/internal/verify"
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

	var instruction string
	if reviewType == "phase" {
		instruction = fmt.Sprintf("Code review and quality checks should have already been completed during the milestone implementation.\nPlease perform a lightweight phase completion check:\n1. Confirm all milestone specs are met\n2. Confirm code is production-ready\n3. Note any follow-up items for upcoming milestones\nWrite your assessment to `.orc_history/HANDOFF-%s.md`.", projectName)
	} else {
		instruction = fmt.Sprintf("Please perform a %s review and provide feedback. Write your review to `.orc_history/HANDOFF-%s.md`.", reviewType, projectName)
	}

	prompt := fmt.Sprintf(
		"Review: %s for %s\n\n## Context Bundle\n\n%s\n\n%s",
		reviewType, target, bundle, instruction,
	)
	return bundle, prompt
}

func Phase(milestoneID, milestoneName, milestoneSpec, workDir string, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	ms := []context.MilestoneInfo{{Name: milestoneName, Spec: milestoneSpec}}
	_, prompt := buildReviewPrompt("phase", milestoneID, ms, handoffNotes, verifyResults)
	result, err := session.Run(milestoneName+"-review", prompt, workDir, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}

func Final(projectName, workDir string, milestones []context.MilestoneInfo, handoffNotes []handoff.Note, verifyResults []verify.Result) (*Result, error) {
	_, prompt := buildReviewPrompt("final", projectName, milestones, handoffNotes, verifyResults)
	result, err := session.Run("final-review-"+projectName, prompt, workDir, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return &Result{
		ReturnCode: result.ReturnCode,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}, nil
}
